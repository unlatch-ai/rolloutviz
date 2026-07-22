package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitWizardInteractiveWritesPreferenceAndConfirmedSkill(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	home := filepath.Join(root, "home")
	configDir := filepath.Join(root, "config")
	input := strings.NewReader("codex,cursor\ny\nn\nn\n")
	var output bytes.Buffer
	result, err := runInitWizard(input, &output, initOptions{Interactive: true, HomeDir: home, ConfigDir: configDir})
	if err != nil {
		t.Fatal(err)
	}
	if result.OpenGallery {
		t.Fatal("gallery opened after no confirmation")
	}
	config, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil || !strings.Contains(string(config), `"open_mode": "browser"`) {
		t.Fatalf("config = %q, %v", config, err)
	}
	skillPath := filepath.Join(home, ".codex", "skills", "rlviz", "SKILL.md")
	skill, err := os.ReadFile(skillPath)
	if err != nil || !strings.Contains(string(skill), "# RLViz trace workflow") || !strings.HasPrefix(string(skill), "---\nname: rlviz") {
		t.Fatalf("skill = %q, %v", skill, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".cursor", "rules", "rlviz.mdc")); !os.IsNotExist(err) {
		t.Fatalf("cursor file should be skipped: %v", err)
	}
	cursorPath := filepath.Join(root, ".cursor", "rules", "rlviz.mdc")
	if !strings.Contains(output.String(), skillPath) || !strings.Contains(output.String(), cursorPath) || !strings.Contains(output.String(), "rlviz inspect <SOURCE>") || !strings.Contains(output.String(), "rlviz plugin trust") {
		t.Fatalf("wizard output missing preview or prompt:\n%s", output.String())
	}
}

func TestCursorRuleDestinationIsAbsoluteAndProjectLocal(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	destination, _, err := initAgentFile("cursor", filepath.Join(root, "home"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".cursor", "rules", "rlviz.mdc")
	if destination != want || !filepath.IsAbs(destination) {
		t.Fatalf("cursor destination = %q, want absolute %q", destination, want)
	}
}

func TestInitWizardYesAndNonTTYNeverReadInput(t *testing.T) {
	for _, test := range []struct {
		name    string
		options initOptions
	}{
		{name: "yes", options: initOptions{Yes: true, Interactive: true}},
		{name: "non-tty", options: initOptions{Interactive: false}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			test.options.ConfigDir = root
			var output bytes.Buffer
			result, err := runInitWizard(failingReader{}, &output, test.options)
			if err != nil {
				t.Fatal(err)
			}
			if result.OpenGallery {
				t.Fatal("unattended setup opened gallery")
			}
			content, err := os.ReadFile(filepath.Join(root, "config.json"))
			if err != nil || !strings.Contains(string(content), `"open_mode": "browser"`) {
				t.Fatalf("default config = %q, %v", content, err)
			}
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, os.ErrPermission }

func TestLoadUserConfigUsesOverride(t *testing.T) {
	t.Setenv("RLVIZ_CONFIG_DIR", t.TempDir())
	path, err := userConfigPath("")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeUserConfig(path, userConfig{SchemaVersion: 1, OpenMode: "both"}); err != nil {
		t.Fatal(err)
	}
	config, exists, err := loadUserConfig()
	if err != nil || !exists || config.OpenMode != "browser" {
		t.Fatalf("config = %#v, exists=%v, err=%v", config, exists, err)
	}
}

func TestRememberLastSourcePreservesInterfacePreference(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("RLVIZ_CONFIG_DIR", configDir)
	source := filepath.Join(t.TempDir(), "trace.ndjson")
	if err := os.WriteFile(source, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := userConfig{SchemaVersion: 1, OpenMode: "both"}
	if err := rememberLastSource(config, true, source); err != nil {
		t.Fatal(err)
	}
	loaded, exists, err := loadUserConfig()
	if err != nil || !exists {
		t.Fatalf("load config: exists=%v err=%v", exists, err)
	}
	if loaded.OpenMode != "browser" || loaded.LastSource != source {
		t.Fatalf("config = %#v", loaded)
	}
	if !usableSource(loaded.LastSource) {
		t.Fatal("remembered source should be usable")
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	if usableSource(loaded.LastSource) {
		t.Fatal("removed source should fall back to samples")
	}
}
