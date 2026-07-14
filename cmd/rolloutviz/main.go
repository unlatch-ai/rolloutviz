package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

const version = "0.0.0-dev"

type result struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func main() {
	command := "help"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "version":
		flags := flag.NewFlagSet("version", flag.ExitOnError)
		jsonOutput := flags.Bool("json", false, "print machine-readable output")
		if err := flags.Parse(os.Args[2:]); err != nil {
			os.Exit(2)
		}
		printVersion(*jsonOutput)
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command)
		os.Exit(2)
	}
}

func printVersion(jsonOutput bool) {
	value := result{Name: "rolloutviz", Version: version}
	if jsonOutput {
		if err := json.NewEncoder(os.Stdout).Encode(value); err != nil {
			fmt.Fprintf(os.Stderr, "encode version output: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("rolloutviz %s\n", value.Version)
}

func printHelp() {
	fmt.Print(`RolloutViz

Visualize and compare agent rollouts.

Usage:
  rolloutviz version [--json]
  rolloutviz help

The open and plugin commands are specified but not implemented yet.
`)
}
