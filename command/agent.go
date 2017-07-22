package command

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/advanderveer/factory/engine"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/hashicorp/logutils"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
)

//Agent command
type Agent struct {
	*command

	awsFlags   AWSFlags
	debugFlags DebugFlags
}

//AgentFactory creates the command
func AgentFactory() cli.CommandFactory {
	cmd := &Agent{}
	cmd.command = createCommand(cmd.Execute, cmd.Description, cmd.Usage)
	cmd.command.flagParser.AddGroup("AWS Flags", "AWS Flags", &cmd.awsFlags)
	cmd.command.flagParser.AddGroup("Debug Flags", "Debug Flags", &cmd.debugFlags)

	return func() (cli.Command, error) {
		return cmd, nil
	}
}

//Execute runs the command
func (cmd *Agent) Execute(args []string) (err error) {
	if len(args) < 1 {
		return errors.New("not enough arguments, see --help")
	}

	awsopts := session.Options{}
	if cmd.awsFlags.Profile != "" {
		awsopts.Profile = cmd.awsFlags.Profile
	}

	if cmd.awsFlags.Region != "" {
		awsopts.Config = aws.Config{Region: aws.String(cmd.awsFlags.Region)}
	}

	var awss *session.Session
	if awss, err = session.NewSessionWithOptions(awsopts); err != nil {
		return errors.Wrap(err, "failed to create aws session")
	}

	logs := log.New(os.Stderr, "factory/", log.Lshortfile)
	logs.SetOutput(&logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel(cmd.debugFlags.Verbosity),
		Writer:   os.Stderr,
	})

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt)
	ctx := context.Background()
	ctx, stop := context.WithTimeout(ctx, time.Second*30)
	defer stop()
	go func() {
		for s := range sigCh {
			logs.Printf("[INFO] Received %s, shutting down", s)
			stop()
		}
	}()

	db := dynamodb.New(awss)
	engine := engine.New(logs, db)
	if err = engine.RunAgent(ctx, args[0]); err != nil {
		return errors.Wrap(err, "failed to run agent")
	}

	return nil
}

// Description returns long-form help text
func (cmd *Agent) Description() string { return "<help>" }

// Synopsis returns a one-line
func (cmd *Agent) Synopsis() string { return "<synopsis>" }

// Usage shows usage
func (cmd *Agent) Usage() string { return "factory agent <pool_id>" }
