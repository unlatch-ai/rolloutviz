package shape_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/TheSnakeFang/rlviz/internal/model"
	"github.com/TheSnakeFang/rlviz/internal/shape"
)

func event(sequence int64, kind string) shape.Event {
	return shape.Event{Sequence: sequence, Kind: kind}
}

func TestSummarizePlacesMidTraceErrorTruthfully(t *testing.T) {
	events := []shape.Event{event(0, "message"), event(25, "tool"), event(50, "error"), event(75, "message"), event(100, "reward")}
	summary := shape.Summarize(events, 48)
	if got := summary.Slots[24].Landmark; got != shape.LandmarkError {
		t.Fatalf("midpoint landmark = %q, want error", got)
	}
	if got := summary.Slots[47].Landmark; got != shape.LandmarkEvidence {
		t.Fatalf("last landmark = %q, want evidence", got)
	}
}

func TestSummarizeLandmarkPriority(t *testing.T) {
	tests := []struct {
		name   string
		events []shape.Event
		want   shape.Landmark
	}{
		{"evidence", []shape.Event{{Sequence: 0, Kind: "grader"}}, shape.LandmarkEvidence},
		{"context over evidence", []shape.Event{{Sequence: 0, Kind: "grader"}, {Sequence: 1, Kind: "state", HasContext: true}}, shape.LandmarkContext},
		{"error over context and evidence", []shape.Event{{Sequence: 0, Kind: "grader"}, {Sequence: 1, Kind: "state", HasContext: true}, {Sequence: 2, Kind: "error", AlignmentKey: "context:compaction"}}, shape.LandmarkError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			summary := shape.Summarize(test.events, 1)
			if got := summary.Slots[0].Landmark; got != test.want {
				t.Fatalf("landmark = %q, want %q", got, test.want)
			}
			if summary.Slots[0].Count != len(test.events) {
				t.Fatalf("count = %d, want %d", summary.Slots[0].Count, len(test.events))
			}
		})
	}
}

func TestSummarizeEmptyAndSingleEvent(t *testing.T) {
	empty := shape.Summarize(nil, 48)
	if empty.Events != 0 || len(empty.Slots) != 48 {
		t.Fatalf("empty summary = %#v", empty)
	}
	single := shape.Summarize([]shape.Event{{Sequence: 99, Kind: "environment_action"}}, 48)
	if single.Events != 1 || single.Slots[0].Count != 1 || single.Slots[0].Tools != 1 {
		t.Fatalf("single summary = %#v", single)
	}
}

func TestSummarizeClampsOutOfRangeSequences(t *testing.T) {
	summary := shape.Summarize([]shape.Event{event(0, "message"), event(-10, "tool"), event(110, "tool"), event(100, "message")}, 4)
	if summary.Slots[0].Count != 2 || summary.Slots[0].Tools != 1 {
		t.Fatalf("first slot = %#v", summary.Slots[0])
	}
	if summary.Slots[3].Count != 2 || summary.Slots[3].Tools != 1 {
		t.Fatalf("last slot = %#v", summary.Slots[3])
	}
}

func TestGalleryShapeSnapshot(t *testing.T) {
	fixture, err := os.Open("../../examples/gallery/coding-agent-bugfix.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	events := make([]shape.Event, 0, 300)
	if err := model.Decode(fixture, func(record *model.Record) error {
		if value, ok := record.Value.(*model.Event); ok && value.TrajectoryID == "coding-bugfix-rollout-01" {
			events = append(events, shape.Event{Sequence: value.Sequence, Kind: value.Kind, AlignmentKey: value.AlignmentKey, HasContext: value.Context != nil})
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(shape.Summarize(events, shape.DefaultSlotCount))
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := os.ReadFile("../../fixtures/shape/coding-agent-bugfix.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot shape.Summary
	if err := json.Unmarshal(wantJSON, &snapshot); err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("gallery shape snapshot differs; regenerate fixtures/shape/coding-agent-bugfix.json\ngot: %s", got)
	}
}
