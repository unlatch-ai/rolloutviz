package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/unlatch-ai/rlviz/integrations"
)

const agentSetupSchemaVersion = "1"

type agentSetupResult struct {
	SchemaVersion        string `json:"schema_version"`
	Command              string `json:"command"`
	Mode                 string `json:"mode"`
	Agent                string `json:"agent"`
	Source               string `json:"source"`
	SuggestedDestination string `json:"suggested_destination"`
	Content              string `json:"content"`
}

func runSetup(arguments []string) {
	if len(arguments) == 0 || arguments[0] == "help" || arguments[0] == "-h" || arguments[0] == "--help" {
		printSetupHelp()
		return
	}
	if arguments[0] != "agent" {
		fmt.Fprintf(os.Stderr, "unknown setup command %q\n", arguments[0])
		printSetupHelpTo(os.Stderr)
		os.Exit(2)
	}
	runSetupAgent(arguments[1:])
}

func runSetupAgent(arguments []string) {
	flags := flag.NewFlagSet("setup agent", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	printOutput := flags.Bool("print", false, "print bundled instructions without writing files")
	jsonOutput := flags.Bool("json", false, "print machine-readable output")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: rlviz setup agent <codex|claude-code|cursor> --print [--json]")
	}
	if err := flags.Parse(normalizeSetupAgentArguments(arguments)); err != nil {
		os.Exit(2)
	}
	if flags.NArg() != 1 || !*printOutput {
		flags.Usage()
		os.Exit(2)
	}

	result, err := loadAgentSetup(flags.Arg(0))
	if err != nil {
		fatalError("setup_agent", *jsonOutput, err)
	}
	if *jsonOutput {
		writeOutput(result, true, "")
		return
	}
	fmt.Print(result.Content)
}

func loadAgentSetup(name string) (agentSetupResult, error) {
	setup, err := integrations.Agent(name)
	if err != nil {
		return agentSetupResult{}, err
	}
	return agentSetupResult{
		SchemaVersion:        agentSetupSchemaVersion,
		Command:              "setup_agent",
		Mode:                 "print",
		Agent:                setup.Agent,
		Source:               setup.Source,
		SuggestedDestination: setup.SuggestedDestination,
		Content:              setup.Content,
	}, nil
}

func normalizeSetupAgentArguments(arguments []string) []string {
	flags := make([]string, 0, len(arguments))
	positional := make([]string, 0, 1)
	for _, argument := range arguments {
		if argument == "--print" || argument == "--json" || strings.HasPrefix(argument, "--print=") || strings.HasPrefix(argument, "--json=") {
			flags = append(flags, argument)
		} else {
			positional = append(positional, argument)
		}
	}
	return append(flags, positional...)
}

func printSetupHelp() {
	printSetupHelpTo(os.Stdout)
}

func printSetupHelpTo(output *os.File) {
	fmt.Fprint(output, `RLViz setup

Print version-matched coding-agent instructions. This command never writes project files.

Usage:
  rlviz setup agent <codex|claude-code|cursor> --print [--json]
`)
}
