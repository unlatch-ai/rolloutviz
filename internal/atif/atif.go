// Package atif maps Harbor's public Agent Trajectory Interchange Format to
// RLViz canonical records. It intentionally knows nothing about Harbor job
// directories, evaluator outputs, or organization-specific result files.
package atif

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TheSnakeFang/rlviz/internal/model"
)

const Format = "harbor-atif-json"

var supportedVersions = map[string]struct{}{
	"ATIF-v1.5": {},
	"ATIF-v1.6": {},
	"ATIF-v1.7": {},
}

// Detect reports whether document has a supported ATIF schema version and the
// required root shape. It is deliberately strict so arbitrary JSON is not
// silently interpreted as a trajectory.
func Detect(document map[string]any) bool {
	version, _ := document["schema_version"].(string)
	_, versionOK := supportedVersions[version]
	_, agentOK := document["agent"].(map[string]any)
	_, stepsOK := document["steps"].([]any)
	return versionOK && agentOK && stepsOK
}

// Probe performs bounded-friendly format detection. It returns as soon as the
// required schema_version, agent object, and steps array headers are seen; it
// does not decode the steps themselves when producers use the specified field
// order.
func Probe(reader io.Reader) (bool, string, error) {
	decoder := json.NewDecoder(reader)
	start, err := decoder.Token()
	if err != nil {
		return false, "", err
	}
	if start != json.Delim('{') {
		return false, "", nil
	}
	version := ""
	hasAgent, hasSteps := false, false
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return false, version, err
		}
		key, _ := keyToken.(string)
		switch key {
		case "schema_version":
			if err := decoder.Decode(&version); err != nil {
				return false, "", err
			}
			if _, ok := supportedVersions[version]; !ok {
				return false, version, nil
			}
		case "agent":
			token, err := decoder.Token()
			if err != nil {
				return false, version, err
			}
			delim, ok := token.(json.Delim)
			if !ok || delim != '{' {
				return false, version, nil
			}
			hasAgent = true
			if err := skipOpenValue(decoder, '}'); err != nil {
				return false, version, err
			}
		case "steps":
			token, err := decoder.Token()
			if err != nil {
				return false, version, err
			}
			delim, ok := token.(json.Delim)
			if !ok || delim != '[' {
				return false, version, nil
			}
			hasSteps = true
			if _, ok := supportedVersions[version]; ok && hasAgent {
				return true, version, nil
			}
			if err := skipOpenValue(decoder, ']'); err != nil {
				return false, version, err
			}
		default:
			if err := skipValue(decoder); err != nil {
				return false, version, err
			}
		}
		if _, ok := supportedVersions[version]; ok && hasAgent && hasSteps {
			return true, version, nil
		}
	}
	return false, version, nil
}

func skipValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	closing := byte('}')
	if delim == '[' {
		closing = ']'
	}
	return skipOpenValue(decoder, closing)
}

func skipOpenValue(decoder *json.Decoder, closing byte) error {
	for decoder.More() {
		if err := skipValue(decoder); err != nil {
			return err
		}
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim(closing) {
		return fmt.Errorf("malformed JSON: expected %q", closing)
	}
	return nil
}

// NormalizeBytes validates and normalizes one complete ATIF JSON document.
func NormalizeBytes(source []byte, sourceName string) ([]byte, error) {
	var document map[string]any
	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode ATIF JSON: %w", err)
	}
	return Normalize(document, sourceName)
}

// Normalize converts one ATIF document, including v1.7 embedded subagents, to
// deterministic canonical NDJSON.
func Normalize(document map[string]any, sourceName string) ([]byte, error) {
	if !Detect(document) {
		return nil, errors.New("unsupported ATIF document: expected ATIF-v1.5, ATIF-v1.6, or ATIF-v1.7")
	}
	rootKey := firstNonempty(stringValue(document["trajectory_id"]), stringValue(document["session_id"]), sourceName)
	runID := stableID("atif-run", stringValue(document["session_id"]), rootKey)
	caseID := stableID("atif-case", runID, rootKey)
	groupID := stableID("atif-group", caseID)

	var out bytes.Buffer
	count := int64(0)
	emit := func(value any) error {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		out.Write(encoded)
		out.WriteByte('\n')
		count++
		return nil
	}

	agent := object(document["agent"])
	runMetadata := model.Metadata{
		"source_format": stringValue(document["schema_version"]),
		"agent":         agent,
	}
	if extra := object(document["extra"]); extra != nil {
		runMetadata["extra"] = extra
	}
	if err := emit(model.Run{RecordType: model.RecordRun, ID: runID, Name: firstNonempty(stringValue(agent["name"]), rootKey), Metadata: runMetadata}); err != nil {
		return nil, err
	}
	if err := emit(model.Case{RecordType: model.RecordCase, ID: caseID, RunID: runID, Name: rootKey, Input: firstUserMessage(document), Metadata: model.Metadata{"session_id": document["session_id"]}}); err != nil {
		return nil, err
	}
	if err := emit(model.Group{RecordType: model.RecordGroup, ID: groupID, CaseID: caseID, Name: "ATIF trajectory"}); err != nil {
		return nil, err
	}

	seenTrajectoryIDs := map[string]bool{}
	var emitTrajectory func(map[string]any, string, int) error
	emitTrajectory = func(trajectory map[string]any, parentID string, sibling int) error {
		version := stringValue(trajectory["schema_version"])
		if _, ok := supportedVersions[version]; !ok {
			return fmt.Errorf("embedded trajectory %d has unsupported schema_version %q", sibling, version)
		}
		if object(trajectory["agent"]) == nil || array(trajectory["steps"]) == nil {
			return fmt.Errorf("ATIF trajectory %q requires agent and steps", stringValue(trajectory["trajectory_id"]))
		}
		nativeID := firstNonempty(stringValue(trajectory["trajectory_id"]), stringValue(trajectory["session_id"]), fmt.Sprintf("trajectory-%d", sibling))
		trajectoryID := stableID("atif-trajectory", runID, parentID, nativeID, sibling)
		if seenTrajectoryIDs[trajectoryID] {
			return fmt.Errorf("duplicate ATIF trajectory identifier %q", nativeID)
		}
		seenTrajectoryIDs[trajectoryID] = true
		metadata := model.Metadata{
			"source_format": version,
			"session_id":    trajectory["session_id"],
			"trajectory_id": trajectory["trajectory_id"],
			"agent":         trajectory["agent"],
			"notes":         trajectory["notes"],
		}
		if continuation := stringValue(trajectory["continued_trajectory_ref"]); continuation != "" {
			metadata["continued_trajectory_ref"] = continuation
		}
		if err := emit(model.Trajectory{RecordType: model.RecordTrajectory, ID: trajectoryID, GroupID: groupID, ParentID: parentID, Termination: stringValue(trajectory["continued_trajectory_ref"]), Metadata: compact(metadata)}); err != nil {
			return err
		}

		sequence := int64(0)
		parentEventID := ""
		for index, rawStep := range array(trajectory["steps"]) {
			step := object(rawStep)
			if step == nil {
				return fmt.Errorf("ATIF trajectory %q step %d must be an object", nativeID, index+1)
			}
			stepID, ok := integer(step["step_id"])
			if !ok || stepID < 0 {
				return fmt.Errorf("ATIF trajectory %q step %d requires a nonnegative step_id", nativeID, index+1)
			}
			source := stringValue(step["source"])
			if source != "system" && source != "user" && source != "agent" {
				return fmt.Errorf("ATIF trajectory %q step %d has unsupported source %q", nativeID, stepID, source)
			}
			message := step["message"]
			if message != nil && (source != "agent" || hasContent(message) || len(array(step["tool_calls"])) == 0) {
				kind := "message"
				role := source
				if source == "agent" {
					kind, role = "generation", "assistant"
				}
				eventID := stableID("atif-event", trajectoryID, stepID, "message")
				data := model.Metadata{"role": role, "content": message, "reasoning_content": step["reasoning_content"], "metrics": step["metrics"], "llm_call_count": step["llm_call_count"], "is_copied_context": step["is_copied_context"]}
				eventMetadata := model.Metadata{"atif_step_id": stepID, "model_name": step["model_name"], "reasoning_effort": step["reasoning_effort"]}
				if tokens, ok := tokenTotal(object(step["metrics"])); ok {
					eventMetadata["token_count"] = tokens
				}
				if cost := object(step["metrics"])["cost_usd"]; cost != nil {
					eventMetadata["cost_usd"] = cost
				}
				event := model.Event{RecordType: model.RecordEvent, ID: eventID, TrajectoryID: trajectoryID, Sequence: sequence, Kind: kind, ParentID: parentEventID, Timestamp: stringValue(step["timestamp"]), AlignmentKey: kind + ":" + role, Output: map[string]any{"role": role, "content": message}, Data: compact(data), Source: &model.SourceLocation{Path: sourceName}, Metadata: compact(eventMetadata)}
				if raw, err := json.Marshal(step); err == nil {
					event.Raw = raw
				}
				if err := emit(event); err != nil {
					return err
				}
				if err := emitImageArtifacts(emit, trajectoryID, eventID, message, sourceName); err != nil {
					return err
				}
				parentEventID, sequence = eventID, sequence+1
			}

			resultsByCall := map[string][]map[string]any{}
			unmatched := make([]map[string]any, 0)
			if observation := object(step["observation"]); observation != nil {
				for _, rawResult := range array(observation["results"]) {
					result := object(rawResult)
					if result == nil {
						continue
					}
					callID := stringValue(result["source_call_id"])
					if callID == "" {
						unmatched = append(unmatched, result)
					} else {
						resultsByCall[callID] = append(resultsByCall[callID], result)
					}
				}
			}
			for toolIndex, rawCall := range array(step["tool_calls"]) {
				call := object(rawCall)
				if call == nil {
					return fmt.Errorf("ATIF trajectory %q step %d tool call %d must be an object", nativeID, stepID, toolIndex+1)
				}
				callID := stringValue(call["tool_call_id"])
				toolName := stringValue(call["function_name"])
				if callID == "" || toolName == "" || object(call["arguments"]) == nil {
					return fmt.Errorf("ATIF trajectory %q step %d tool call %d requires tool_call_id, function_name, and object arguments", nativeID, stepID, toolIndex+1)
				}
				eventID := stableID("atif-event", trajectoryID, stepID, "tool", callID)
				event := model.Event{RecordType: model.RecordEvent, ID: eventID, TrajectoryID: trajectoryID, Sequence: sequence, Kind: "tool", ParentID: parentEventID, Timestamp: stringValue(step["timestamp"]), AlignmentKey: "tool:" + toolName, Input: map[string]any{"id": callID, "name": toolName, "arguments": call["arguments"]}, Data: compact(model.Metadata{"extra": call["extra"]}), Source: &model.SourceLocation{Path: sourceName}, Metadata: model.Metadata{"atif_step_id": stepID, "tool_call_id": callID, "title": toolName}}
				if raw, err := json.Marshal(call); err == nil {
					event.Raw = raw
				}
				if err := emit(event); err != nil {
					return err
				}
				parentEventID, sequence = eventID, sequence+1
				for resultIndex, result := range resultsByCall[callID] {
					observationID := stableID("atif-event", trajectoryID, stepID, "observation", callID, resultIndex)
					observation := model.Event{RecordType: model.RecordEvent, ID: observationID, TrajectoryID: trajectoryID, Sequence: sequence, Kind: "observation", ParentID: parentEventID, Timestamp: stringValue(step["timestamp"]), AlignmentKey: "observation:" + toolName, Output: result["content"], Data: compact(model.Metadata{"source_call_id": callID, "subagent_trajectory_ref": result["subagent_trajectory_ref"], "extra": result["extra"]}), Source: &model.SourceLocation{Path: sourceName}, Metadata: model.Metadata{"atif_step_id": stepID, "tool_name": toolName, "title": toolName + " result"}}
					if raw, err := json.Marshal(result); err == nil {
						observation.Raw = raw
					}
					if err := emit(observation); err != nil {
						return err
					}
					if err := emitImageArtifacts(emit, trajectoryID, observationID, result["content"], sourceName); err != nil {
						return err
					}
					parentEventID, sequence = observationID, sequence+1
				}
				delete(resultsByCall, callID)
			}
			for _, values := range resultsByCall {
				unmatched = append(unmatched, values...)
			}
			for resultIndex, result := range unmatched {
				eventID := stableID("atif-event", trajectoryID, stepID, "observation", "unmatched", resultIndex)
				event := model.Event{RecordType: model.RecordEvent, ID: eventID, TrajectoryID: trajectoryID, Sequence: sequence, Kind: "observation", ParentID: parentEventID, Timestamp: stringValue(step["timestamp"]), AlignmentKey: "observation", Output: result["content"], Data: compact(model.Metadata{"source_call_id": result["source_call_id"], "subagent_trajectory_ref": result["subagent_trajectory_ref"], "extra": result["extra"]}), Source: &model.SourceLocation{Path: sourceName}, Metadata: model.Metadata{"atif_step_id": stepID}}
				if raw, err := json.Marshal(result); err == nil {
					event.Raw = raw
				}
				if err := emit(event); err != nil {
					return err
				}
				if err := emitImageArtifacts(emit, trajectoryID, eventID, result["content"], sourceName); err != nil {
					return err
				}
				parentEventID, sequence = eventID, sequence+1
			}
		}

		metricNames := make([]string, 0)
		for name, value := range object(trajectory["final_metrics"]) {
			if name != "extra" && scalar(value) {
				metricNames = append(metricNames, name)
			}
		}
		sort.Strings(metricNames)
		for _, name := range metricNames {
			if err := emit(model.Signal{RecordType: model.RecordSignal, ID: stableID("atif-signal", trajectoryID, name), TrajectoryID: trajectoryID, Name: name, Value: object(trajectory["final_metrics"])[name], Unit: metricUnit(name), Metadata: model.Metadata{"provenance": "source_native"}}); err != nil {
				return err
			}
		}
		if total, ok := tokenTotal(object(trajectory["final_metrics"])); ok {
			if err := emit(model.Signal{RecordType: model.RecordSignal, ID: stableID("atif-signal", trajectoryID, "token_count"), TrajectoryID: trajectoryID, Name: "token_count", Value: total, Unit: "count", Metadata: model.Metadata{"provenance": "adapter_derived", "derivation": "total_prompt_tokens + total_completion_tokens"}}); err != nil {
				return err
			}
		}

		childIDs := map[string]bool{}
		for childIndex, rawChild := range array(trajectory["subagent_trajectories"]) {
			child := object(rawChild)
			if child == nil {
				return fmt.Errorf("ATIF trajectory %q embedded subagent %d must be an object", nativeID, childIndex+1)
			}
			childID := stringValue(child["trajectory_id"])
			if childID == "" {
				return fmt.Errorf("ATIF-v1.7 embedded subagent %d requires trajectory_id", childIndex+1)
			}
			if childIDs[childID] {
				return fmt.Errorf("ATIF-v1.7 embedded subagent trajectory_id %q is duplicated", childID)
			}
			childIDs[childID] = true
			if err := emitTrajectory(child, trajectoryID, childIndex); err != nil {
				return err
			}
		}
		return nil
	}

	if err := emitTrajectory(document, "", 0); err != nil {
		return nil, err
	}
	if err := emit(model.Complete{RecordType: model.RecordComplete, Records: count, Warnings: 0}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func firstUserMessage(document map[string]any) any {
	for _, raw := range array(document["steps"]) {
		step := object(raw)
		if stringValue(step["source"]) == "user" {
			return step["message"]
		}
	}
	return nil
}

func emitImageArtifacts(emit func(any) error, trajectoryID, eventID string, content any, sourceName string) error {
	for index, rawPart := range array(content) {
		part := object(rawPart)
		if stringValue(part["type"]) != "image" {
			continue
		}
		source := object(part["source"])
		path := stringValue(source["path"])
		mediaType := stringValue(source["media_type"])
		if path == "" || mediaType == "" {
			continue
		}
		artifact := model.Artifact{RecordType: model.RecordArtifact, ID: stableID("atif-artifact", trajectoryID, eventID, index, path), TrajectoryID: trajectoryID, EventID: eventID, Name: filepath.Base(path), MediaType: mediaType, Path: path, Metadata: model.Metadata{"source_document": sourceName, "provenance": "source_native"}}
		if err := emit(artifact); err != nil {
			return err
		}
	}
	return nil
}

func object(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

func array(value any) []any {
	result, _ := value.([]any)
	return result
}

func stringValue(value any) string {
	result, _ := value.(string)
	return result
}

func integer(value any) (int64, bool) {
	if valueNumber, ok := value.(json.Number); ok {
		result, err := valueNumber.Int64()
		return result, err == nil
	}
	valueFloat, ok := value.(float64)
	if !ok || math.IsNaN(valueFloat) || math.IsInf(valueFloat, 0) || math.Trunc(valueFloat) != valueFloat {
		return 0, false
	}
	return int64(valueFloat), true
}

func hasContent(value any) bool {
	switch item := value.(type) {
	case string:
		return strings.TrimSpace(item) != ""
	case []any:
		return len(item) > 0
	default:
		return item != nil
	}
}

func compact(metadata model.Metadata) model.Metadata {
	result := model.Metadata{}
	for key, value := range metadata {
		if value != nil && value != "" {
			result[key] = value
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func scalar(value any) bool {
	switch value.(type) {
	case bool, string, float64, json.Number:
		return true
	default:
		return false
	}
}

func metricUnit(name string) string {
	if strings.HasSuffix(name, "_tokens") || name == "total_steps" {
		return "count"
	}
	if strings.HasSuffix(name, "_usd") {
		return "USD"
	}
	return ""
}

func tokenTotal(metrics map[string]any) (int64, bool) {
	if metrics == nil {
		return 0, false
	}
	prompt, promptOK := integer(metrics["prompt_tokens"])
	if !promptOK {
		prompt, promptOK = integer(metrics["total_prompt_tokens"])
	}
	completion, completionOK := integer(metrics["completion_tokens"])
	if !completionOK {
		completion, completionOK = integer(metrics["total_completion_tokens"])
	}
	if !promptOK && !completionOK {
		return 0, false
	}
	return prompt + completion, true
}

func firstNonempty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "atif"
}

func stableID(prefix string, parts ...any) string {
	encoded, _ := json.Marshal(parts)
	sum := sha256.Sum256(encoded)
	return prefix + "-" + hex.EncodeToString(sum[:12])
}
