// Package sourceprofile extracts a bounded, value-free structural summary from
// a source file. It is intended for adapter onboarding, not data validation.
package sourceprofile

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	KindJSONObject = "json_object"
	KindJSONArray  = "json_array"
	KindNDJSON     = "ndjson"
	KindText       = "text"
	KindBinary     = "binary"
)

// Limits bounds all source reading and structural traversal. Zero-valued
// fields use DefaultLimits.
type Limits struct {
	MaxBytes      int64 `json:"max_bytes"`
	MaxDepth      int   `json:"max_depth"`
	MaxPaths      int   `json:"max_paths"`
	MaxArrayItems int   `json:"max_array_items"`
}

// DefaultLimits deliberately keeps profiling cheap enough to run during
// interactive adapter onboarding.
var DefaultLimits = Limits{
	MaxBytes:      256 << 10,
	MaxDepth:      8,
	MaxPaths:      512,
	MaxArrayItems: 16,
}

// Field describes the JSON types observed at a structural path. Types and
// fields are sorted to make the profile deterministic.
type Field struct {
	Path  string   `json:"path"`
	Types []string `json:"types"`
}

// Profile contains no JSON scalar values. SourceBytes comes from file
// metadata; SampleBytes is the exact number of bytes inspected.
type Profile struct {
	Kind        string  `json:"kind"`
	SourceBytes int64   `json:"source_bytes"`
	SampleBytes int     `json:"sample_bytes"`
	Truncated   bool    `json:"truncated"`
	Limits      Limits  `json:"limits"`
	Fields      []Field `json:"fields,omitempty"`
}

// ProfileFile profiles a regular file. Directories and other non-regular
// filesystem objects are rejected rather than read implicitly.
func ProfileFile(path string, limits Limits) (Profile, error) {
	limits, err := normalizeLimits(limits)
	if err != nil {
		return Profile{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Profile{}, err
	}
	if !info.Mode().IsRegular() {
		return Profile{}, fmt.Errorf("source profile requires a regular file: %s", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return Profile{}, err
	}
	defer f.Close()

	// The extra byte proves truncation without allowing an unbounded read.
	sample, err := io.ReadAll(io.LimitReader(f, limits.MaxBytes+1))
	if err != nil {
		return Profile{}, err
	}
	truncated := int64(len(sample)) > limits.MaxBytes
	if truncated {
		sample = sample[:limits.MaxBytes]
	}

	p := Profile{
		SourceBytes: info.Size(),
		SampleBytes: len(sample),
		Truncated:   truncated,
		Limits:      limits,
	}
	p.Kind, p.Fields = classify(sample, truncated, limits)
	return p, nil
}

func normalizeLimits(got Limits) (Limits, error) {
	result := got
	if result.MaxBytes == 0 {
		result.MaxBytes = DefaultLimits.MaxBytes
	}
	if result.MaxDepth == 0 {
		result.MaxDepth = DefaultLimits.MaxDepth
	}
	if result.MaxPaths == 0 {
		result.MaxPaths = DefaultLimits.MaxPaths
	}
	if result.MaxArrayItems == 0 {
		result.MaxArrayItems = DefaultLimits.MaxArrayItems
	}
	if result.MaxBytes < 1 || result.MaxDepth < 1 || result.MaxPaths < 1 || result.MaxArrayItems < 1 {
		return Limits{}, errors.New("source profile limits must all be positive")
	}
	return result, nil
}

func classify(sample []byte, truncated bool, limits Limits) (string, []Field) {
	if isBinary(sample) {
		return KindBinary, nil
	}
	trimmed := bytes.TrimSpace(sample)
	if len(trimmed) == 0 {
		return KindText, nil
	}

	collector := newCollector(limits)
	root, complete := collectJSON(trimmed, "$", collector)
	if complete && root == "object" {
		return KindJSONObject, collector.fields()
	}
	if complete && root == "array" {
		return KindJSONArray, collector.fields()
	}

	collector = newCollector(limits)
	if collectNDJSON(sample, truncated, collector) {
		return KindNDJSON, collector.fields()
	}

	// A capped sample can end halfway through one large JSON document. Preserve
	// the useful structure already decoded, while never treating malformed,
	// fully-read input as JSON.
	if truncated && (trimmed[0] == '{' || trimmed[0] == '[') {
		collector = newCollector(limits)
		root, _ = collectJSON(trimmed, "$", collector)
		if root == "object" {
			return KindJSONObject, collector.fields()
		}
		if root == "array" {
			return KindJSONArray, collector.fields()
		}
	}

	return KindText, nil
}

func isBinary(data []byte) bool {
	if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return true
	}
	if len(data) == 0 {
		return false
	}
	controls := 0
	for _, b := range data {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' {
			controls++
		}
	}
	return controls*20 > len(data) // More than 5% non-text controls.
}

func collectNDJSON(data []byte, truncated bool, collector *pathCollector) bool {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Lines remain bounded by MaxBytes, but Scanner's default 64 KiB token cap
	// is too small for otherwise valid sampled records.
	scanner.Buffer(make([]byte, 4096), len(data)+1)
	lines := make([][]byte, 0, collector.limits.MaxArrayItems)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) != 0 {
			lines = append(lines, bytes.Clone(line))
		}
	}
	if scanner.Err() != nil || len(lines) < 2 {
		return false
	}

	lastComplete := len(data) == 0 || data[len(data)-1] == '\n' || data[len(data)-1] == '\r'
	valid := 0
	for i, line := range lines {
		probe := newCollector(collector.limits)
		root, complete := collectJSON(line, "$[]", probe)
		if !complete || (root != "object" && root != "array") {
			if truncated && i == len(lines)-1 && !lastComplete {
				break
			}
			return false
		}
		valid++
		if valid <= collector.limits.MaxArrayItems {
			collector.merge(probe)
		}
	}
	return valid >= 2
}

type pathCollector struct {
	limits Limits
	types  map[string]map[string]struct{}
}

func newCollector(limits Limits) *pathCollector {
	return &pathCollector{limits: limits, types: make(map[string]map[string]struct{})}
}

func (c *pathCollector) add(path, typ string, depth int, enabled bool) {
	if !enabled || depth > c.limits.MaxDepth {
		return
	}
	set, exists := c.types[path]
	if !exists {
		if len(c.types) >= c.limits.MaxPaths {
			return
		}
		set = make(map[string]struct{})
		c.types[path] = set
	}
	set[typ] = struct{}{}
}

func (c *pathCollector) merge(other *pathCollector) {
	fields := other.fields()
	for _, field := range fields {
		for _, typ := range field.Types {
			c.add(field.Path, typ, 0, true)
		}
	}
}

func (c *pathCollector) fields() []Field {
	paths := make([]string, 0, len(c.types))
	for path := range c.types {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	fields := make([]Field, 0, len(paths))
	for _, path := range paths {
		types := make([]string, 0, len(c.types[path]))
		for typ := range c.types[path] {
			types = append(types, typ)
		}
		sort.Strings(types)
		fields = append(fields, Field{Path: path, Types: types})
	}
	return fields
}

func collectJSON(data []byte, base string, collector *pathCollector) (string, bool) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	root, err := collectValue(decoder, base, 0, collector, true)
	if err != nil {
		return root, false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return root, false
	}
	return root, true
}

func collectValue(decoder *json.Decoder, path string, depth int, collector *pathCollector, enabled bool) (string, error) {
	token, err := decoder.Token()
	if err != nil {
		return "", err
	}
	switch value := token.(type) {
	case json.Delim:
		switch value {
		case '{':
			collector.add(path, "object", depth, enabled)
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return "object", err
				}
				key, ok := keyToken.(string)
				if !ok {
					return "object", errors.New("JSON object key is not a string")
				}
				if _, err := collectValue(decoder, childPath(path, key), depth+1, collector, enabled && depth < collector.limits.MaxDepth); err != nil {
					return "object", err
				}
			}
			_, err = decoder.Token()
			return "object", err
		case '[':
			collector.add(path, "array", depth, enabled)
			index := 0
			for decoder.More() {
				collectItem := enabled && index < collector.limits.MaxArrayItems && depth < collector.limits.MaxDepth
				if _, err := collectValue(decoder, path+"[]", depth+1, collector, collectItem); err != nil {
					return "array", err
				}
				index++
			}
			_, err = decoder.Token()
			return "array", err
		default:
			return "", fmt.Errorf("unexpected JSON delimiter %q", value)
		}
	case nil:
		collector.add(path, "null", depth, enabled)
		return "null", nil
	case bool:
		collector.add(path, "boolean", depth, enabled)
		return "boolean", nil
	case json.Number:
		collector.add(path, "number", depth, enabled)
		return "number", nil
	case string:
		collector.add(path, "string", depth, enabled)
		return "string", nil
	default:
		return "", fmt.Errorf("unexpected JSON token type %T", token)
	}
}

func childPath(parent, key string) string {
	if isSimpleKey(key) {
		return parent + "." + key
	}
	encoded, _ := json.Marshal(key)
	return parent + "[" + string(encoded) + "]"
}

func isSimpleKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if !(r == '_' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return !strings.ContainsAny(key, ".[]")
}
