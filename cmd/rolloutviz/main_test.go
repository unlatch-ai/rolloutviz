package main

import (
	"reflect"
	"testing"
)

func TestNormalizeViewerArgumentsAllowsFlagsAfterPath(t *testing.T) {
	got := normalizeViewerArguments([]string{"trace.ndjson", "--no-open", "--port", "7317", "--json"})
	want := []string{"--no-open", "--port", "7317", "--json", "trace.ndjson"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeViewerArguments() = %#v, want %#v", got, want)
	}
}

func TestNormalizeViewerArgumentsPreservesEqualsFlag(t *testing.T) {
	got := normalizeViewerArguments([]string{"trace.ndjson", "--port=7317"})
	want := []string{"--port=7317", "trace.ndjson"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeViewerArguments() = %#v, want %#v", got, want)
	}
}
