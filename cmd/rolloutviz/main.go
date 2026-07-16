package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/unlatch-ai/rolloutviz/internal/app"
)

const version = "0.0.0-dev"

type result struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type serveResult struct {
	URL     string `json:"url"`
	Path    string `json:"path"`
	Command string `json:"command"`
	Mode    string `json:"mode"`
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
	case "open", "serve":
		runViewer(command, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command)
		os.Exit(2)
	}
}

func runViewer(command string, arguments []string) {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	noOpen := flags.Bool("no-open", command == "serve", "do not open the browser")
	port := flags.Int("port", 0, "loopback port (0 selects an available port)")
	jsonOutput := flags.Bool("json", false, "print machine-readable startup output")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: rlviz %s [--no-open] [--port PORT] [--json] PATH\n", command)
	}
	if err := flags.Parse(normalizeViewerArguments(arguments)); err != nil {
		os.Exit(2)
	}
	if flags.NArg() != 1 {
		flags.Usage()
		os.Exit(2)
	}

	viewer, err := app.StartViewer(app.Viewer{SourcePath: flags.Arg(0), Port: *port})
	if err != nil {
		writeError(command, *jsonOutput, err)
		os.Exit(1)
	}
	output := serveResult{URL: viewer.URL, Path: viewer.SourcePath, Command: command, Mode: "foreground"}
	if *jsonOutput {
		if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
			fmt.Fprintf(os.Stderr, "encode startup output: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("RolloutViz is serving %s at %s (foreground; press Ctrl-C to stop)\n", output.Path, output.URL)
	}
	if !*noOpen {
		if err := openBrowser(viewer.URL); err != nil {
			fmt.Fprintf(os.Stderr, "open browser: %v\n", err)
		}
	}
	if err := viewer.Serve(); err != nil {
		writeError(command, *jsonOutput, err)
		os.Exit(1)
	}
}

// normalizeViewerArguments lets coding agents place flags before or after the
// source path, while retaining flag.FlagSet's validation and error messages.
func normalizeViewerArguments(arguments []string) []string {
	flags := make([]string, 0, len(arguments))
	paths := make([]string, 0, 1)
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		switch argument {
		case "--port":
			flags = append(flags, argument)
			if index+1 < len(arguments) {
				index++
				flags = append(flags, arguments[index])
			}
		case "--no-open", "--json":
			flags = append(flags, argument)
		default:
			if len(argument) > 2 && argument[:2] == "--" {
				flags = append(flags, argument)
			} else {
				paths = append(paths, argument)
			}
		}
	}
	return append(flags, paths...)
}

func writeError(command string, jsonOutput bool, err error) {
	if jsonOutput {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"code": command + "_failed", "error": err.Error()})
		return
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", command, err)
}

func openBrowser(url string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", url)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	return command.Start()
}

func printVersion(jsonOutput bool) {
	value := result{Name: "rlviz", Version: version}
	if jsonOutput {
		if err := json.NewEncoder(os.Stdout).Encode(value); err != nil {
			fmt.Fprintf(os.Stderr, "encode version output: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("rlviz %s\n", value.Version)
}

func printHelp() {
	fmt.Print(`RolloutViz

Visualize and compare agent rollouts.

Usage:
  rlviz version [--json]
  rlviz open [--no-open] [--port PORT] [--json] PATH
  rlviz serve [--no-open] [--port PORT] [--json] PATH
  rlviz help

The open and serve commands run in the foreground in this initial release.
`)
}
