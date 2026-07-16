package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSourceResolvesRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trajectory.jsonl")
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ValidateSource(path)
	if err != nil {
		t.Fatalf("ValidateSource() error = %v", err)
	}
	want, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ValidateSource() = %q, want %q", got, want)
	}
}

func TestValidateSourceRejectsDirectory(t *testing.T) {
	if _, err := ValidateSource(t.TempDir()); err == nil {
		t.Fatal("ValidateSource() accepted a directory")
	}
}

func TestValidateSourceRejectsMissingPath(t *testing.T) {
	if _, err := ValidateSource(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("ValidateSource() accepted a missing path")
	}
}
