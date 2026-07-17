package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverInventoriesWithoutExecutingAndRanksDeterministically(t *testing.T) {
	root := t.TempDir()
	sentinel := filepath.Join(root, "executed")
	for _, name := range []string{"zeta", "alpha"} {
		dir := filepath.Join(root, name)
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeDiscoveryPlugin(t, dir, name, "#!/bin/sh\ntouch "+sentinel+"\n")
	}
	result := Discover(DiscoveryOptions{Roots: []DiscoveryRoot{{Path: root, Source: "project"}}})
	if len(result.Issues) != 0 || len(result.Plugins) != 2 {
		t.Fatalf("result = %#v", result)
	}
	if result.Plugins[0].Name != "alpha" || result.Plugins[0].Rank != 1 || result.Plugins[0].Status != "untrusted" || result.Plugins[1].Name != "zeta" {
		t.Fatalf("plugins = %#v", result.Plugins)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("discovery executed plugin command: %v", err)
	}
}

func TestDiscoverReportsTrustedChangedInvalidAndSkipsSymlinks(t *testing.T) {
	root := t.TempDir()
	currentDir := filepath.Join(root, "current")
	changedDir := filepath.Join(root, "changed")
	invalidDir := filepath.Join(root, "invalid")
	for _, dir := range []string{currentDir, changedDir, invalidDir} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeDiscoveryPlugin(t, currentDir, "current", "#!/bin/sh\nexit 0\n")
	writeDiscoveryPlugin(t, changedDir, "changed", "#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(filepath.Join(invalidDir, ManifestName), []byte("not: a: manifest\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	current, err := Load(currentDir)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := Load(changedDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changedDir, "helper.txt"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	writeDiscoveryPlugin(t, outside, "outside", "#!/bin/sh\nexit 0\n")
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}

	result := Discover(DiscoveryOptions{
		Roots:        []DiscoveryRoot{{Path: root, Source: "project"}},
		TrustEntries: []TrustEntry{{Path: currentDir, Digest: current.Digest}, {Path: changedDir, Digest: changed.Digest}},
	})
	statuses := map[string]string{}
	for _, item := range result.Plugins {
		statuses[filepath.Base(item.Path)] = item.Status
	}
	if statuses["current"] != "trusted" || statuses["changed"] != "changed" || statuses["invalid"] != "invalid" {
		t.Fatalf("statuses = %#v; result = %#v", statuses, result)
	}
	if _, ok := statuses["linked"]; ok {
		t.Fatalf("followed symlink: %#v", result.Plugins)
	}
}

func TestDiscoverBoundsDepthSizeAndCountFailClosed(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "one", "two", "three")
	if err := os.MkdirAll(deep, 0o700); err != nil {
		t.Fatal(err)
	}
	writeDiscoveryPlugin(t, deep, "too-deep", "#!/bin/sh\n")
	large := filepath.Join(root, "large")
	if err := os.Mkdir(large, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(large, ManifestName), []byte(strings.Repeat("x", 65)), 0o600); err != nil {
		t.Fatal(err)
	}

	result := Discover(DiscoveryOptions{Roots: []DiscoveryRoot{{Path: root, Source: "project"}}, MaxDepth: 2, MaxManifestBytes: 64})
	if len(result.Plugins) != 1 || result.Plugins[0].Status != "invalid" || !strings.Contains(result.Plugins[0].Error, "exceeds") {
		t.Fatalf("bounded result = %#v", result)
	}

	for _, name := range []string{"a", "b"} {
		dir := filepath.Join(t.TempDir(), name)
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeDiscoveryPlugin(t, dir, name, "#!/bin/sh\n")
		// Each supplied directory is itself a root containing one manifest.
		result = Discover(DiscoveryOptions{Roots: []DiscoveryRoot{{Path: dir, Source: "explicit"}}, MaxPlugins: 1})
		if len(result.Plugins) != 1 {
			t.Fatalf("single result = %#v", result)
		}
	}

	countRoot := t.TempDir()
	for _, name := range []string{"a", "b"} {
		dir := filepath.Join(countRoot, name)
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeDiscoveryPlugin(t, dir, name, "#!/bin/sh\n")
	}
	result = Discover(DiscoveryOptions{Roots: []DiscoveryRoot{{Path: countRoot, Source: "project"}}, MaxPlugins: 1})
	if len(result.Plugins) != 0 || len(result.Issues) != 1 || result.Issues[0].Code != "plugin_limit" {
		t.Fatalf("limit must fail closed: %#v", result)
	}
}

func TestDiscoverRejectsSymlinkRoot(t *testing.T) {
	realRoot := t.TempDir()
	link := filepath.Join(t.TempDir(), "plugins")
	if err := os.Symlink(realRoot, link); err != nil {
		t.Fatal(err)
	}
	result := Discover(DiscoveryOptions{Roots: []DiscoveryRoot{{Path: link, Source: "project"}}})
	if len(result.Plugins) != 0 || len(result.Issues) != 1 || result.Issues[0].Code != "root_invalid" {
		t.Fatalf("result = %#v", result)
	}
}

func writeDiscoveryPlugin(t *testing.T, dir, name, script string) {
	t.Helper()
	manifest := "api_version: rlviz.dev/v1alpha1\nkind: Adapter\nname: " + name + "\nversion: 1.0.0\ncommand:\n  - ./adapter.sh\ncapabilities:\n  - adapter.probe\n  - adapter.stream\n"
	if err := os.WriteFile(filepath.Join(dir, ManifestName), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "adapter.sh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
}
