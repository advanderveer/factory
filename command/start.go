package command

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/posener/complete"
)

//Start command
type Start struct {
	*command

	awsFlags   AWSFlags
	debugFlags DebugFlags
}

//StartFactory creates the command
func StartFactory() cli.CommandFactory {
	cmd := &Start{}
	cmd.command = createCommand(cmd.Execute, cmd.Description, cmd.Usage)
	cmd.command.flagParser.AddGroup("AWS Flags", "AWS Flags", &cmd.awsFlags)
	cmd.command.flagParser.AddGroup("Debug Flags", "Debug Flags", &cmd.debugFlags)

	return func() (cli.Command, error) {
		return cmd, nil
	}
}

//Execute runs the command
func (cmd *Start) Execute(args []string) (err error) {
	var awss *session.Session
	if awss, err = session.NewSessionWithOptions(session.Options{
		Profile: cmd.awsFlags.Profile,
	}); err != nil {
		return errors.Wrap(err, "failed to create aws session")
	}

	fmt.Println(args)
	fmt.Printf("flags: %#v\n", cmd.awsFlags)
	_ = awss

	return nil
}

// Description returns long-form help text
func (cmd *Start) Description() string { return "<help>" }

// Synopsis returns a one-line
func (cmd *Start) Synopsis() string { return "<synopsis>" }

// Usage shows usage
func (cmd *Start) Usage() string { return "<usage>" }

// AutocompleteArgs returns the argument predictor for this command.
func (cmd *Start) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}
