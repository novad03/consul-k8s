package main

import (
	"os"

	"github.com/hashicorp/consul-k8s/subcommand"
	"github.com/mitchellh/cli"
)

// Commands is the mapping of all available consul-k8s commands.
var Commands map[string]cli.CommandFactory

func init() {
	ui := &cli.BasicUi{Writer: os.Stdout, ErrorWriter: os.Stderr}

	Commands = map[string]cli.CommandFactory{
		"inject": func() (cli.Command, error) {
			return &subcommand.Inject{UI: ui}, nil
		},
	}
}
