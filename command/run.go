package command

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/advanderveer/factory/engine"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
)

//Run command
type Run struct {
	*command

	awsFlags   AWSFlags
	debugFlags DebugFlags
}

//RunFactory creates the command
func RunFactory() cli.CommandFactory {
	cmd := &Run{}
	cmd.command = createCommand(cmd.Execute, cmd.Description, cmd.Usage)
	cmd.command.flagParser.AddGroup("AWS Flags", "AWS Flags", &cmd.awsFlags)
	cmd.command.flagParser.AddGroup("Debug Flags", "Debug Flags", &cmd.debugFlags)

	return func() (cli.Command, error) {
		return cmd, nil
	}
}

//Execute runs the command
func (cmd *Run) Execute(args []string) (err error) {
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

	logs := cmd.debugFlags.Logger()
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
	q := sqs.New(awss)
	engine := engine.New(logs, db, q)
	err = engine.Run(ctx, args[0], 1)
	if err != nil {
		return errors.Wrap(err, "failed to run process")
	}

	return nil
}

// Description returns long-form help text
func (cmd *Run) Description() string { return "<help>" }

// Synopsis returns a one-line
func (cmd *Run) Synopsis() string { return "<synopsis>" }

// Usage shows usage
func (cmd *Run) Usage() string { return "factory run <pool_id>" }
