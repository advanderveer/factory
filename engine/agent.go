package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/advanderveer/factory/model"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/pkg/errors"
)

var (
	//MaxAgentShutdownTime determines how long the agent gets for a shutdown
	MaxAgentShutdownTime = time.Second * 5

	//AgentHeartbeatInterval determines how often the agent reports home
	AgentHeartbeatInterval = time.Second * 10

	//ExecutorRunTimeout determines how long the message handler waits for the executor to accept a run message
	ExecutorRunTimeout = DockerRunExecTimeout + (5 * time.Second)
)

//HandleNodeMessage will start handling node messages
func (e *Engine) HandleNodeMessage(ctx context.Context, nodePK model.NodePK, doneCh chan<- struct{}, runCh chan<- RunMsg) {
	e.logs.Printf("[INFO] Start handling messages for node '%s'", nodePK)
	defer e.logs.Printf("[INFO] Stopped handling messages for node '%s'", nodePK)
	defer close(doneCh)

	for {
		if err := NextNodeMessage(ctx, e.q, nodePK, func(nextMsg string) bool {

			e.logs.Printf("[DEBUG] Received run message: '%s'", nextMsg)
			msg := RunMsg{}
			err := json.Unmarshal([]byte(nextMsg), &msg)
			if err != nil {
				e.logs.Printf("[ERROR] Failed to unmarshal run message: %v", err)
				return false
			}

			select {
			case <-time.After(ExecutorRunTimeout):
				e.logs.Printf("[ERROR] Timed out waiting for executor to accept message '%s'", nextMsg)
				return false
			case runCh <- msg:
			}

			return true
		}); err != nil {
			if aerr, ok := errors.Cause(err).(awserr.Error); ok && aerr.Code() == request.CanceledErrorCode {
				e.logs.Printf("[INFO] Mext node message receive was cancelled")
				return
			}

			e.logs.Printf("[ERROR] Failed to receive next node message: %v", err)
			return
		}

	}
}

func (e *Engine) shutdownAgent(node *model.Node, handleDoneCh chan struct{}, execDoneCh chan struct{}) error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, MaxAgentShutdownTime)
	defer cancel()

	err := e.deleteNode(ctx, node.NodePK)
	if err != nil {
		return errors.Wrap(err, "failed to delete node")
	}

	e.logs.Printf("[INFO] Waiting for handling routine to exit")
	select {
	case <-handleDoneCh:
	case <-ctx.Done():
		return errors.Wrap(err, "handling routine didn't exit in time")
	}

	e.logs.Printf("[INFO] Waiting for executor routine to exit")
	select {
	case <-handleDoneCh:
	case <-ctx.Done():
		return errors.Wrap(err, "executor routine didn't exit in time")
	}

	return nil
}

//Agent will start the node agent
func (e *Engine) Agent(ctx context.Context, poolID string) (err error) {
	e.logs.Printf("[INFO] Starting node agent for pool '%s'", poolID)
	defer e.logs.Printf("[INFO] Exited node agent")

	node, err := model.RegisterNode(ctx, e.db, poolID, time.Now().Add(2*AgentHeartbeatInterval))
	if err != nil {
		return errors.Wrap(err, "failed to register node")
	}

	e.logs.Printf("[DEBUG] Creating queue for node '%s'", node.NodePK)
	err = CreateNodeQueue(ctx, e.q, node.NodePK)
	if err != nil {
		return errors.Wrap(err, "failed to create node queue")
	}

	exec, err := NewDockerExec(e.logs, e.db)
	if err != nil {
		return errors.Wrap(err, "failed to create docker executer")
	}

	go exec.Start(ctx, node.NodeID)

	handleMsgDoneCh := make(chan struct{})
	go e.HandleNodeMessage(ctx, node.NodePK, handleMsgDoneCh, exec.Incoming)

	ticker := time.NewTicker(AgentHeartbeatInterval)
	for {
		select {
		case <-ctx.Done():
			return e.shutdownAgent(node, handleMsgDoneCh, exec.Done)
		case <-ticker.C:
			t := 2 * AgentHeartbeatInterval
			e.logs.Printf("[DEBUG] Incrementing node Heartbeat (+%s)", t)
			err := model.IncrementNodeTTL(ctx, e.db, node.NodePK, t)
			if err != nil {
				if errors.Cause(err) == model.ErrNodeNotExists {
					e.logs.Printf("[INFO] Node entry removed, shutting down")
					return nil
				}

				return errors.Wrap(err, "failed to increment node ttl")
			}
		}
	}
}
