// Package shape computes compact, truth-positioned trajectory summaries.
package shape

import "strings"

const DefaultSlotCount = 48

type Landmark string

const (
	LandmarkError    Landmark = "error"
	LandmarkContext  Landmark = "context"
	LandmarkEvidence Landmark = "evidence"
)

// Event is the minimal canonical event projection needed for a shape summary.
type Event struct {
	Sequence     int64
	Kind         string
	AlignmentKey string
	HasContext   bool
}

type Slot struct {
	Count    int      `json:"count"`
	Tools    int      `json:"tools"`
	Landmark Landmark `json:"landmark,omitempty"`
}

type Summary struct {
	Events int    `json:"events"`
	Slots  []Slot `json:"slots"`
}

// Summarize mirrors web/src/instrument.ts summarizeShape. Events are placed on
// their sequence axis between the first and last input events.
func Summarize(events []Event, slotCount int) Summary {
	if slotCount < 0 {
		slotCount = 0
	}
	summary := Summary{Events: len(events), Slots: make([]Slot, slotCount)}
	if len(events) == 0 || slotCount == 0 {
		return summary
	}
	first, last := events[0].Sequence, events[len(events)-1].Sequence
	span := float64(last - first)
	if span < 1e-9 {
		span = 1e-9
	}
	for _, event := range events {
		index := int((float64(event.Sequence-first) / span) * float64(slotCount))
		if index < 0 {
			index = 0
		} else if index >= slotCount {
			index = slotCount - 1
		}
		slot := &summary.Slots[index]
		slot.Count++
		if event.Kind == "tool" || event.Kind == "environment_action" {
			slot.Tools++
		}
		landmark := landmarkFor(event)
		if landmarkPriority(landmark) > landmarkPriority(slot.Landmark) {
			slot.Landmark = landmark
		}
	}
	return summary
}

func landmarkFor(event Event) Landmark {
	if event.Kind == "error" {
		return LandmarkError
	}
	if event.HasContext || strings.HasPrefix(event.AlignmentKey, "context:") {
		return LandmarkContext
	}
	if event.Kind == "reward" || event.Kind == "grader" {
		return LandmarkEvidence
	}
	return ""
}

func landmarkPriority(landmark Landmark) int {
	switch landmark {
	case LandmarkError:
		return 3
	case LandmarkContext:
		return 2
	case LandmarkEvidence:
		return 1
	default:
		return 0
	}
}
