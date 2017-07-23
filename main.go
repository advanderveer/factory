package main

import (
	"fmt"
	"log"
	"os"

	"github.com/advanderveer/factory/command"
	"github.com/mitchellh/cli"
)

var (
	version string
	commit  string
)

func main() {
	logs := log.New(os.Stderr, "factory/", log.Lshortfile)

	c := &cli.CLI{
		Name:         "factory",
		Version:      fmt.Sprintf("%s (%s)", version, commit),
		Args:         os.Args[1:],
		Autocomplete: true,
		Commands: map[string]cli.CommandFactory{
			"pump":  command.PumpFactory(),
			"agent": command.AgentFactory(),
			"run":   command.RunFactory(),
			"evict": command.EvictFactory(),
		},
	}

	status, err := c.Run()
	if err != nil {
		logs.Println(err)
	}

	os.Exit(status)
}
