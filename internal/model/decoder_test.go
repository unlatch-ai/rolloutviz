package model

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalFixtures(t *testing.T) {
	t.Parallel()
	patterns := []string{
		filepath.Join("..", "..", "fixtures", "canonical", "*.ndjson"),
		filepath.Join("..", "..", "fixtures", "adversarial", "*.ndjson"),
	}
	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) == 0 {
			t.Fatalf("no fixtures matched %s", pattern)
		}
		for _, path := range files {
			path := path
			t.Run(filepath.Base(path), func(t *testing.T) {
				t.Parallel()
				file, err := os.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				var records int
				if err := Decode(file, func(record *Record) error {
					records++
					if len(record.Raw) == 0 {
						t.Fatal("raw record was not retained")
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
				if records < 2 {
					t.Fatalf("decoded only %d records", records)
				}
			})
		}
	}
}

func TestMalformedFixtures(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"unknown-parent.ndjson":     "unknown or later parent",
		"out-of-order.ndjson":       "not greater than prior sequence",
		"duplicate-id.ndjson":       "duplicate id",
		"complete-not-final.ndjson": "complete must be the final record",
		"unknown-field.ndjson":      "unknown field",
		"invalid-json.ndjson":       "invalid JSON",
	}
	for name, want := range tests {
		name, want := name, want
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			file, err := os.Open(filepath.Join("..", "..", "fixtures", "malformed", name))
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			err = Decode(file, nil)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("Decode() error = %v, want substring %q", err, want)
			}
		})
	}
}

func TestDecoderStreamsLargeRecords(t *testing.T) {
	t.Parallel()
	payload := strings.Repeat("x", 256*1024)
	stream := strings.Join([]string{
		`{"record_type":"run","id":"run-large","metadata":{"payload":"` + payload + `"}}`,
		`{"record_type":"complete","records":1,"warnings":0}`,
		"",
	}, "\n")
	decoder := NewDecoder(strings.NewReader(stream))
	record, err := decoder.Next()
	if err != nil {
		t.Fatal(err)
	}
	if record.Type != RecordRun {
		t.Fatalf("type = %q, want run", record.Type)
	}
	if len(record.Raw) < len(payload) {
		t.Fatalf("raw record unexpectedly short: %d", len(record.Raw))
	}
	if _, err := decoder.Next(); err != nil {
		t.Fatal(err)
	}
	if _, err := decoder.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("final error = %v, want EOF", err)
	}
}

func TestDecodeRequiresComplete(t *testing.T) {
	t.Parallel()
	err := Decode(strings.NewReader(`{"record_type":"run","id":"run-incomplete"}`+"\n"), nil)
	if err == nil || !strings.Contains(err.Error(), "without a complete") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodePropagatesVisitorError(t *testing.T) {
	t.Parallel()
	want := errors.New("stop")
	err := Decode(strings.NewReader(`{"record_type":"run","id":"run-stop"}`+"\n"), func(*Record) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}
