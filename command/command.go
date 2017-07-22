package command

import (
	"bytes"
	"fmt"
	"os"
	"reflect"

	flags "github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"github.com/posener/complete"
)

type command struct {
	flagParser *flags.Parser
	runFunc    func(args []string) error
	helpFunc   func() string
}

func createCommand(runFunc func([]string) error, helpFunc func() string, usageFunc func() string) *command {
	return &command{
		flags.NewNamedParser(usageFunc(), flags.None),
		runFunc,
		helpFunc,
	}
}

func addFlagPredicts(fl complete.Flags, f *flags.Option) {
	if f.Field().Type.Kind() == reflect.Bool || f.Field().Type == reflect.SliceOf(reflect.TypeOf(true)) {
		fl["--"+f.LongName] = complete.PredictNothing
		if f.ShortName != 0 {
			fl[fmt.Sprintf("-%s", string(f.ShortName))] = complete.PredictNothing
		}

	} else {
		fl["--"+f.LongName] = complete.PredictAnything
		if f.ShortName != 0 {
			fl[fmt.Sprintf("-%s", string(f.ShortName))] = complete.PredictAnything
		}
	}
}

// AutocompleteFlags returns a mapping of supported flags
func (cmd *command) AutocompleteFlags() (fl complete.Flags) {
	fl = complete.Flags{}
	for _, g := range cmd.flagParser.Groups() {
		for _, f := range g.Options() {
			addFlagPredicts(fl, f)
		}
	}

	return fl
}

//Help shows extensive help
func (cmd *command) Help() string {
	buf := bytes.NewBuffer(nil)
	cmd.flagParser.WriteHelp(buf)
	return fmt.Sprintf(`
%s
%s`, cmd.helpFunc(), buf.String())
}

//Run runs the actual command
func (cmd *command) Run(args []string) int {
	remaining, err := cmd.flagParser.ParseArgs(args)
	if err != nil {
		return fail(err, "failed to parse flags(s)")
	}

	err = cmd.runFunc(remaining)
	if err != nil {
		return fail(err, "failed to run")
	}

	return 0
}

// AutocompleteArgs returns the argument predictor for this command.
func (cmd *command) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func fail(err error, message string) int {
	fmt.Fprintf(os.Stderr, "error: %v\n", errors.Wrap(err, message))
	return 255
}
