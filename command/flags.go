package command

import (
	"log"
	"os"

	"github.com/hashicorp/logutils"
)

//AWSFlags holds options that configure aws
type AWSFlags struct {
	Profile string `long:"aws-profile" description:"AWS Credentials Profile"`
	Region  string `long:"aws-region" description:"AWS Region"`
}

//DebugFlags are used to get more insight into the program behaviour
type DebugFlags struct {
	Debug     bool   `long:"debug" description:"Debug mode enable extra information"`
	Verbosity string `short:"v" long:"verbosity" default:"DEBUG"  description:"Show information at various levels: DEBUG, INFO, WARN, ERROR"`
}

//Logger returns a logger that filters based on verbosity flags
func (f DebugFlags) Logger() (logs *log.Logger) {
	logs = log.New(os.Stderr, "factory/", log.Lshortfile|log.Lmicroseconds)
	logs.SetOutput(&logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel(f.Verbosity),
		Writer:   os.Stderr,
	})

	return logs
}
