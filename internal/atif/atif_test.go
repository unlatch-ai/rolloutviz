package atif

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/TheSnakeFang/rlviz/internal/model"
)

func TestNormalizeSyntheticATIF(t *testing.T) {
	data, err := os.ReadFile("../../examples/traces/harbor-atif.json")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if !Detect(document) {
		t.Fatal("synthetic fixture was not detected")
	}
	canonical, err := Normalize(document, "harbor-atif.json")
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := Normalize(document, "harbor-atif.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(canonical, repeated) {
		t.Fatal("normalization is not deterministic")
	}

	trajectories, events, signals, artifacts := 0, 0, 0, 0
	if err := model.Decode(bytes.NewReader(canonical), func(record *model.Record) error {
		switch record.Value.(type) {
		case *model.Trajectory:
			trajectories++
		case *model.Event:
			events++
		case *model.Signal:
			signals++
		case *model.Artifact:
			artifacts++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if trajectories != 2 || events != 6 || signals != 6 || artifacts != 0 {
		t.Fatalf("records: trajectories=%d events=%d signals=%d artifacts=%d", trajectories, events, signals, artifacts)
	}
}

func TestDetectRejectsUnknownATIFVersion(t *testing.T) {
	document := map[string]any{"schema_version": "ATIF-v2.0", "agent": map[string]any{}, "steps": []any{}}
	if Detect(document) {
		t.Fatal("unknown ATIF version was detected")
	}
}

func TestNormalizeAcceptsHistoricalZeroBasedStepIDs(t *testing.T) {
	document := map[string]any{
		"schema_version": "ATIF-v1.6",
		"session_id":     "zero-based",
		"agent":          map[string]any{"name": "agent", "version": "1"},
		"steps":          []any{map[string]any{"step_id": float64(0), "source": "user", "message": "start"}},
	}
	if _, err := Normalize(document, "trajectory.json"); err != nil {
		t.Fatal(err)
	}
}
