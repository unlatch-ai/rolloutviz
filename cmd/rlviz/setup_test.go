package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLoadAgentSetup(t *testing.T) {
	tests := []struct {
		name        string
		destination string
		marker      string
	}{
		{name: "codex", destination: "AGENTS.md", marker: "# RLViz trace workflow"},
		{name: "claude-code", destination: "CLAUDE.md", marker: "# RLViz trace workflow"},
		{name: "cursor", destination: ".cursor/rules/rlviz.mdc", marker: "alwaysApply: false"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := loadAgentSetup(test.name)
			if err != nil {
				t.Fatalf("loadAgentSetup() error = %v", err)
			}
			if result.SchemaVersion != "1" || result.Command != "setup_agent" || result.Mode != "print" {
				t.Fatalf("unexpected stable metadata: %#v", result)
			}
			if result.Agent != test.name || result.SuggestedDestination != test.destination {
				t.Fatalf("unexpected agent metadata: %#v", result)
			}
			if result.Source == "" || !strings.Contains(result.Content, test.marker) {
				t.Fatalf("missing bundled instructions: %#v", result)
			}
		})
	}
}

func TestLoadAgentSetupRejectsUnknownAgent(t *testing.T) {
	_, err := loadAgentSetup("other")
	if err == nil || !strings.Contains(err.Error(), "choose codex, claude-code, or cursor") {
		t.Fatalf("loadAgentSetup() error = %v", err)
	}
}

func TestAgentSetupJSONContract(t *testing.T) {
	result, err := loadAgentSetup("codex")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	wantFields := []string{
		`"schema_version":"1"`,
		`"command":"setup_agent"`,
		`"mode":"print"`,
		`"agent":"codex"`,
		`"source":"integrations/codex/AGENTS.md"`,
		`"suggested_destination":"AGENTS.md"`,
		`"content":`,
	}
	for _, field := range wantFields {
		if !strings.Contains(string(encoded), field) {
			t.Fatalf("JSON %s does not contain %s", encoded, field)
		}
	}
}

func TestNormalizeSetupAgentArguments(t *testing.T) {
	got := normalizeSetupAgentArguments([]string{"codex", "--print", "--json"})
	want := []string{"--print", "--json", "codex"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("normalizeSetupAgentArguments() = %q, want %q", got, want)
	}
}
