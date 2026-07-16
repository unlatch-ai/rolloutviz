package model

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type Decoder struct {
	reader *bufio.Reader
	line   int64
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{reader: bufio.NewReader(r)}
}

// Next decodes one NDJSON record without buffering the entire stream.
func (d *Decoder) Next() (*Record, error) {
	line, err := d.reader.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(line) == 0 && errors.Is(err, io.EOF) {
		return nil, io.EOF
	}
	d.line++
	line = bytes.TrimSuffix(line, []byte{'\n'})
	line = bytes.TrimSuffix(line, []byte{'\r'})
	if len(bytes.TrimSpace(line)) == 0 {
		return nil, fmt.Errorf("line %d: blank records are not allowed", d.line)
	}

	var envelope struct {
		RecordType RecordType `json:"record_type"`
	}
	if decodeErr := strictDecode(line, &envelope); decodeErr != nil {
		// The envelope intentionally accepts unknown fields. Decode it normally,
		// then apply strict decoding to the selected concrete record below.
		if jsonErr := json.Unmarshal(line, &envelope); jsonErr != nil {
			return nil, fmt.Errorf("line %d: invalid JSON: %w", d.line, jsonErr)
		}
	}
	if envelope.RecordType == "" {
		return nil, fmt.Errorf("line %d: record_type is required", d.line)
	}

	var value any
	switch envelope.RecordType {
	case RecordRun:
		value = &Run{}
	case RecordCase:
		value = &Case{}
	case RecordGroup:
		value = &Group{}
	case RecordTrajectory:
		value = &Trajectory{}
	case RecordEvent:
		value = &Event{}
	case RecordSignal:
		value = &Signal{}
	case RecordArtifact:
		value = &Artifact{}
	case RecordComplete:
		value = &Complete{}
	default:
		return nil, fmt.Errorf("line %d: unsupported record_type %q", d.line, envelope.RecordType)
	}
	if decodeErr := strictDecode(line, value); decodeErr != nil {
		return nil, fmt.Errorf("line %d: invalid %s record: %w", d.line, envelope.RecordType, decodeErr)
	}

	raw := append(json.RawMessage(nil), line...)
	return &Record{Type: envelope.RecordType, Value: value, Raw: raw, Line: d.line}, nil
}

func strictDecode(data []byte, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values in one record")
		}
		return err
	}
	return nil
}

// Decode validates and visits a stream record by record.
func Decode(r io.Reader, visit func(*Record) error) error {
	decoder := NewDecoder(r)
	validator := NewValidator()
	for {
		record, err := decoder.Next()
		if errors.Is(err, io.EOF) {
			return validator.Finish()
		}
		if err != nil {
			return err
		}
		if err := validator.Add(record); err != nil {
			return fmt.Errorf("line %d: %w", record.Line, err)
		}
		if visit != nil {
			if err := visit(record); err != nil {
				return err
			}
		}
	}
}
