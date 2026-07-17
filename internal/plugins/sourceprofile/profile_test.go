package sourceprofile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProfileFileJSONObjectIsValueFreeAndDeterministic(t *testing.T) {
	path := writeFixture(t, `{
  "run_id": "VALUE_DO_NOT_EXPOSE_7f3c",
  "score": 0.875,
  "ok": true,
  "nullable": null,
  "steps": [
    {"role": "user", "content": "ANOTHER_PRIVATE_VALUE"},
    {"role": "assistant", "content": "PRIVATE_RESPONSE", "usage": {"tokens": 42}}
  ]
}`)

	first, err := ProfileFile(path, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := ProfileFile(path, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("profiles differ:\n%#v\n%#v", first, second)
	}
	if first.Kind != KindJSONObject || first.Truncated {
		t.Fatalf("unexpected classification: %#v", first)
	}

	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"VALUE_DO_NOT_EXPOSE_7f3c", "ANOTHER_PRIVATE_VALUE", "PRIVATE_RESPONSE", "0.875", "42"} {
		if strings.Contains(string(encoded), value) {
			t.Fatalf("profile exposed scalar value %q: %s", value, encoded)
		}
	}
	want := map[string]string{
		"$":                      "object",
		"$.nullable":             "null",
		"$.ok":                   "boolean",
		"$.run_id":               "string",
		"$.score":                "number",
		"$.steps":                "array",
		"$.steps[]":              "object",
		"$.steps[].content":      "string",
		"$.steps[].role":         "string",
		"$.steps[].usage":        "object",
		"$.steps[].usage.tokens": "number",
	}
	assertFields(t, first.Fields, want)
}

func TestProfileFileJSONArrayAndSamplingBounds(t *testing.T) {
	path := writeFixture(t, `[
  {"common": 1, "first": true},
  {"common": "two", "second": true},
  {"outside_sample": "must not shape profile"}
]`)
	p, err := ProfileFile(path, Limits{MaxBytes: 1024, MaxDepth: 3, MaxPaths: 10, MaxArrayItems: 2})
	if err != nil {
		t.Fatal(err)
	}
	if p.Kind != KindJSONArray {
		t.Fatalf("kind = %q", p.Kind)
	}
	got := fieldsMap(p.Fields)
	if _, exists := got["$[].outside_sample"]; exists {
		t.Fatalf("field beyond array sample was included: %#v", p.Fields)
	}
	if got["$[].common"] != "number,string" {
		t.Fatalf("union types = %q", got["$[].common"])
	}
}

func TestProfileFileNDJSON(t *testing.T) {
	path := writeFixture(t, "{\"id\":\"sensitive-one\",\"reward\":1}\n{\"id\":\"sensitive-two\",\"reward\":null,\"meta\":{\"ok\":true}}\n")
	p, err := ProfileFile(path, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Kind != KindNDJSON {
		t.Fatalf("kind = %q, profile = %#v", p.Kind, p)
	}
	assertFields(t, p.Fields, map[string]string{
		"$[]":         "object",
		"$[].id":      "string",
		"$[].meta":    "object",
		"$[].meta.ok": "boolean",
		"$[].reward":  "null,number",
	})
	encoded, _ := json.Marshal(p)
	if strings.Contains(string(encoded), "sensitive-one") || strings.Contains(string(encoded), "sensitive-two") {
		t.Fatalf("NDJSON profile exposed values: %s", encoded)
	}
}

func TestProfileFileBinaryAndMalformedText(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		kind string
	}{
		{name: "nul", data: []byte{'P', 'N', 'G', 0, 1, 2}, kind: KindBinary},
		{name: "invalid utf8", data: []byte{0xff, 0xfe, 0xfd}, kind: KindBinary},
		{name: "malformed json", data: []byte(`{"secret":"not closed"`), kind: KindText},
		{name: "plain text", data: []byte("rollout log text\nsecond line"), kind: KindText},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ProfileFile(writeBytes(t, tt.data), Limits{})
			if err != nil {
				t.Fatal(err)
			}
			if p.Kind != tt.kind || len(p.Fields) != 0 {
				t.Fatalf("profile = %#v", p)
			}
		})
	}
}

func TestProfileFileTruncatedJSONReportsExactSample(t *testing.T) {
	value := `{"visible":"scalar-must-not-appear","items":[1,2,3,4,5,6,7,8,9],"tail":"` + strings.Repeat("x", 200) + `"}`
	path := writeFixture(t, value)
	p, err := ProfileFile(path, Limits{MaxBytes: 64, MaxDepth: 4, MaxPaths: 20, MaxArrayItems: 3})
	if err != nil {
		t.Fatal(err)
	}
	if p.Kind != KindJSONObject || !p.Truncated || p.SampleBytes != 64 || p.SourceBytes != int64(len(value)) {
		t.Fatalf("profile = %#v", p)
	}
	encoded, _ := json.Marshal(p)
	if strings.Contains(string(encoded), "scalar-must-not-appear") {
		t.Fatalf("truncated profile exposed value: %s", encoded)
	}
	got := fieldsMap(p.Fields)
	if got["$.visible"] != "string" || got["$.items[]"] != "number" {
		t.Fatalf("missing partial structure: %#v", p.Fields)
	}
}

func TestProfileFileBoundsDepthAndPaths(t *testing.T) {
	path := writeFixture(t, `{"a":{"b":{"c":{"d":1}}},"e":1,"f":2,"g":3}`)
	p, err := ProfileFile(path, Limits{MaxBytes: 1024, MaxDepth: 2, MaxPaths: 3, MaxArrayItems: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Fields) != 3 {
		t.Fatalf("path bound not enforced: %#v", p.Fields)
	}
	for _, field := range p.Fields {
		if field.Path == "$.a.b.c" || field.Path == "$.a.b.c.d" {
			t.Fatalf("depth bound not enforced: %#v", p.Fields)
		}
	}
}

func TestProfileFileRejectsInvalidLimitsAndDirectories(t *testing.T) {
	if _, err := ProfileFile(t.TempDir(), Limits{}); err == nil {
		t.Fatal("expected directory error")
	}
	path := writeFixture(t, `{}`)
	if _, err := ProfileFile(path, Limits{MaxBytes: -1}); err == nil {
		t.Fatal("expected invalid limit error")
	}
}

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	return writeBytes(t, []byte(content))
}

func writeBytes(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func fieldsMap(fields []Field) map[string]string {
	result := make(map[string]string, len(fields))
	for _, field := range fields {
		result[field.Path] = strings.Join(field.Types, ",")
	}
	return result
}

func assertFields(t *testing.T, fields []Field, want map[string]string) {
	t.Helper()
	got := fieldsMap(fields)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fields:\n got: %#v\nwant: %#v", got, want)
	}
}
