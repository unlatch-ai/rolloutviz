// Package integrations exposes the coding-agent instructions bundled with RLViz.
package integrations

import (
	"embed"
	"fmt"
)

// AgentSetup is a versioned, read-only coding-agent instruction asset.
type AgentSetup struct {
	Agent                string
	Source               string
	SuggestedDestination string
	Content              string
}

//go:embed codex/AGENTS.md claude-code/CLAUDE.md cursor/rlviz.mdc
var files embed.FS

var agentSetups = map[string]struct {
	source      string
	destination string
}{
	"codex":       {source: "codex/AGENTS.md", destination: "AGENTS.md"},
	"claude-code": {source: "claude-code/CLAUDE.md", destination: "CLAUDE.md"},
	"cursor":      {source: "cursor/rlviz.mdc", destination: ".cursor/rules/rlviz.mdc"},
}

// Agent returns the bundled instructions for a supported coding agent.
func Agent(name string) (AgentSetup, error) {
	metadata, ok := agentSetups[name]
	if !ok {
		return AgentSetup{}, fmt.Errorf("unsupported agent %q (choose codex, claude-code, or cursor)", name)
	}
	content, err := files.ReadFile(metadata.source)
	if err != nil {
		return AgentSetup{}, fmt.Errorf("read bundled %s instructions: %w", name, err)
	}
	return AgentSetup{
		Agent:                name,
		Source:               "integrations/" + metadata.source,
		SuggestedDestination: metadata.destination,
		Content:              string(content),
	}, nil
}
