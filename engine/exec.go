package engine

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/advanderveer/factory/model"
	"github.com/pkg/errors"
)

var (
	//ExecRunningInterval determines at what rate the executor lists running tasks
	ExecRunningInterval = time.Second * 5

	//DefaultDockerExecTimeout is how long the executer will wait on Docker
	DefaultDockerExecTimeout = time.Second

	//DockerRunExecTimeout is how long the executor will wait for docker runs
	DockerRunExecTimeout = time.Second * 10

	//DiscardLines does nothing with each line
	DiscardLines = func(line string) error {
		return nil
	}
)

//DockerExec uses docker binary to exec
type DockerExec struct {
	dpath string
	logs  *log.Logger
	db    model.DB

	Incoming chan RunMsg
	Done     chan struct{}
}

//NewDockerExec will create a Docker executer
func NewDockerExec(logs *log.Logger, db model.DB) (*DockerExec, error) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("failed to find Docker executable in path: %v, is it installed?", err)
	}

	logs.Printf("[DEBUG] using Docker executable '%s'", dockerPath)
	exec := &DockerExec{
		db:       db,
		dpath:    dockerPath,
		logs:     logs,
		Done:     make(chan struct{}),
		Incoming: make(chan RunMsg),
	}

	return exec, nil
}

func (exe *DockerExec) execDockerTimeout(ctx context.Context, to time.Duration, lineHandler func(line string) error, arg ...string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

	cmd := exec.CommandContext(ctx, exe.dpath, arg...)
	rc, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "failed to pipe stdout")
	}

	defer rc.Close()
	scanner := bufio.NewScanner(rc)

	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "failed to start command")
	}

	for scanner.Scan() {
		err = lineHandler(scanner.Text())
		if err != nil {
			return errors.Wrap(err, "failed to handle line")
		}
	}
	if err = scanner.Err(); err != nil {
		return errors.Wrap(err, "failed while scanning command output")
	}

	err = cmd.Wait()
	if err != nil {
		return errors.Wrap(err, "failed to wait for command")
	}

	return nil
}

func (exe *DockerExec) execDocker(ctx context.Context, lineHandler func(line string) error, arg ...string) (err error) {
	return exe.execDockerTimeout(ctx, time.Second, lineHandler, arg...)
}

func (exe *DockerExec) sendHeartbeats(ctx context.Context, nodeID string) error {
	psargs := []string{"container", "ps", "-f", "label=factory.claim", "--format", "{{.ID}}\t{{.Label \"factory.claim\"}}"}
	if err := exe.execDocker(ctx, func(line string) error {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return errors.Errorf("unexpected docker ps line: '%s'", line)
		}

		exe.logs.Printf("[DEBUG] Send heartbeat container: '%s' claim: '%s' as node: '%s'", fields[0], fields[1], nodeID)
		pk := model.ClaimPK{ClaimID: fields[1]}
		err := model.IncrementClaimTTL(ctx, exe.db, pk, nodeID, ClaimHeartbeatTimeout*2)
		if err != nil {
			if errors.Cause(err) == model.ErrClaimNotExists {
				exe.logs.Printf("[INFO] Container '%s' claim '%s' for node '%s' no longer exists, stopping...", fields[0], fields[1], nodeID)

				args := []string{"container", "stop", "-t=10", fields[0]}
				if err = exe.execDockerTimeout(ctx, time.Second*11, DiscardLines, args...); err != nil {
					return errors.Wrapf(err, "failed to run: docker %v", args)
				}

				return nil
			}

			return errors.Wrap(err, "failed to increment claim ttl")
		}

		return nil
	}, psargs...); err != nil {
		return errors.Wrapf(err, "failed to run: docker %v", psargs)
	}

	return nil
}

func (exe *DockerExec) startContainer(ctx context.Context, claimID string) (err error) {
	args := []string{"container", "run", "-d", "-l", "factory.claim=" + claimID, "redis"}
	if err = exe.execDockerTimeout(ctx, DockerRunExecTimeout, func(line string) error {
		exe.logs.Printf("[INFO] Started container '%s' with claim '%s'", line, claimID)
		return nil
	}, args...); err != nil {
		return errors.Wrapf(err, "failed to run: docker %v", args)
	}

	return nil
}

//Start executing incoming tasks and report heartbeats
func (exe *DockerExec) Start(ctx context.Context, nodeID string) {
	exe.logs.Printf("[INFO] Start handling task runs with executor '%T'", exe)
	defer exe.logs.Printf("[INFO] Stopped handling task runs")
	defer close(exe.Done)

	err := exe.execDocker(ctx, func(line string) error {
		exe.logs.Printf("[INFO] Docker server version: %s", line)
		return nil
	}, "version", "--format='{{.Server.Version}}'")
	if err != nil {
		exe.logs.Printf("[ERROR] Failed to get Docker version: %v", err)
		return
	}

	ticker := time.NewTicker(ExecRunningInterval)
	for {
		select {
		case runMsg := <-exe.Incoming:
			exe.logs.Printf("[INFO] Starting task run: %#v", runMsg)
			err := exe.startContainer(ctx, runMsg.ClaimID)
			if err != nil {
				exe.logs.Printf("[ERROR] Failed to start container: %v", err)
				return
			}

		case <-ticker.C:
			err := exe.sendHeartbeats(ctx, nodeID)
			if err != nil {
				exe.logs.Printf("[ERROR] Failed to send heartbeats: %v", err)
				return
			}

		case <-ctx.Done():
			return
		}
	}
}
