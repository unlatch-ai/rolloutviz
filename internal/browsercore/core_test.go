package browsercore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBuiltInBrowserFormats(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path         string
		format       string
		trajectories int
		firstEvents  int
	}{
		{"../../examples/gallery/coding-agent-bugfix.ndjson", "canonical-ndjson", 1, 300},
		{"../../examples/traces/inspect-ai-eval.json", "inspect-ai-eval-log-json-v2", 2, 5},
		{"../../examples/traces/verifiers-generate.json", "prime-verifiers-generate-outputs", 1, 3},
	}
	for _, test := range cases {
		test := test
		t.Run(filepath.Base(test.path), func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := Parse(data, filepath.Base(test.path))
			if err != nil {
				t.Fatal(err)
			}
			collection, err := DecodeCollection(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if collection.Source.Format != test.format {
				t.Fatalf("format = %q", collection.Source.Format)
			}
			repeated, err := Parse(data, filepath.Base(test.path))
			if err != nil {
				t.Fatal(err)
			}
			if string(encoded) != string(repeated) {
				t.Fatal("browser normalization is not deterministic")
			}
			if got := len(collection.Trajectories); got != test.trajectories {
				t.Fatalf("trajectories = %d, want %d", got, test.trajectories)
			}
			for _, trajectory := range collection.Trajectories {
				if len(trajectory.Events) == test.firstEvents {
					return
				}
			}
			t.Fatalf("no trajectory with %d events", test.firstEvents)
		})
	}
}

func TestRecommendedSizeCeilingExceedsGallery(t *testing.T) {
	data, err := os.ReadFile("../../examples/gallery/coding-agent-bugfix.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) >= MaxRecommendedBytes {
		t.Fatalf("300-event gallery fixture is %d bytes", len(data))
	}
}
