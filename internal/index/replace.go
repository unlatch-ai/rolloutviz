package index

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/TheSnakeFang/rlviz/internal/model"
)

// Replace validates and indexes a canonical NDJSON stream in one transaction.
// An invalid or interrupted replacement leaves the prior source untouched.
func (i *Index) Replace(ctx context.Context, source Source, stream io.Reader) (SourceInfo, error) {
	if strings.TrimSpace(source.ID) == "" {
		return SourceInfo{}, errors.New("source id is required")
	}
	if stream == nil {
		return SourceInfo{}, errors.New("source stream is required")
	}
	decoder := model.NewDecoder(stream)
	return i.replaceRecords(ctx, source, func(yield func(*model.Record) error) error {
		for {
			record, err := decoder.NextContext(ctx)
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("decode source: %w", err)
			}
			if err := yield(record); err != nil {
				return err
			}
		}
	})
}

// ReplaceRecords transactionally indexes an already-decoded record stream.
// It lets trusted adapters feed validated output directly into SQLite without
// first materializing a second complete canonical file on disk.
func (i *Index) ReplaceRecords(ctx context.Context, source Source, stream func(func(*model.Record) error) error) (SourceInfo, error) {
	if stream == nil {
		return SourceInfo{}, errors.New("record stream is required")
	}
	return i.replaceRecords(ctx, source, stream)
}

func (i *Index) replaceRecords(ctx context.Context, source Source, stream func(func(*model.Record) error) error) (SourceInfo, error) {
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return SourceInfo{}, fmt.Errorf("begin index replacement: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM sources WHERE id=?`, source.ID); err != nil {
		return SourceInfo{}, fmt.Errorf("clear prior source: %w", err)
	}
	indexedAt := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO sources
    (id,path,adapter,fingerprint,size,mod_time_ns,indexed_at_ns,records,warnings,complete_raw,index_state,index_error)
    VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, source.ID, source.Path, source.Adapter, source.Fingerprint,
		source.Size, encodeTime(source.ModTime), indexedAt.UnixNano(), 0, 0, []byte{}, Indexing, ""); err != nil {
		return SourceInfo{}, fmt.Errorf("insert source metadata: %w", err)
	}

	validator := model.NewValidator()
	var ordinal int64
	var complete *model.Complete
	var completeRaw []byte
	consumeErr := stream(func(record *model.Record) error {
		if record == nil {
			return errors.New("record stream yielded nil record")
		}
		ordinal++
		if err := validator.Add(record); err != nil {
			return fmt.Errorf("line %d: %w", record.Line, err)
		}
		if err := insertRecord(ctx, tx, source, ordinal, record.ByteOffset, record.ByteLength, record); err != nil {
			return fmt.Errorf("index line %d: %w", record.Line, err)
		}
		if value, ok := record.Value.(*model.Complete); ok {
			complete = value
			completeRaw = append([]byte(nil), record.Raw...)
		}
		return nil
	})
	if consumeErr != nil {
		return SourceInfo{}, consumeErr
	}
	if err := validator.Finish(); err != nil {
		return SourceInfo{}, fmt.Errorf("validate source: %w", err)
	}
	if complete == nil {
		return SourceInfo{}, errors.New("validate source: missing complete record")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sources SET records=?,warnings=?,complete_raw=?,index_state=?,index_error='' WHERE id=?`,
		complete.Records, complete.Warnings, completeRaw, IndexComplete, source.ID); err != nil {
		return SourceInfo{}, fmt.Errorf("finalize source metadata: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return SourceInfo{}, fmt.Errorf("commit index replacement: %w", err)
	}
	return SourceInfo{Source: source, IndexedAt: indexedAt, Records: complete.Records,
		Warnings: complete.Warnings, CompleteRaw: completeRaw, IndexState: IndexComplete}, nil
}

func insertRecord(ctx context.Context, tx *sql.Tx, source Source, ordinal, byteOffset, byteLength int64, record *model.Record) error {
	sourceID := source.ID
	id := recordID(record.Value)
	if _, err := tx.ExecContext(ctx, `INSERT INTO records(source_id,ordinal,record_type,record_id,byte_offset,byte_length,raw) VALUES(?,?,?,?,?,?,?)`,
		sourceID, ordinal, string(record.Type), id, byteOffset, byteLength, []byte(record.Raw)); err != nil {
		return err
	}
	switch v := record.Value.(type) {
	case *model.Run:
		_, err := tx.ExecContext(ctx, `INSERT INTO runs VALUES(?,?,?,?,?,?,?,?)`, sourceID, v.ID, v.Name, v.StartedAt, record.Line, byteOffset, byteLength, []byte(record.Raw))
		return err
	case *model.Case:
		_, err := tx.ExecContext(ctx, `INSERT INTO cases VALUES(?,?,?,?,?,?,?,?)`, sourceID, v.ID, v.RunID, v.Name, record.Line, byteOffset, byteLength, []byte(record.Raw))
		return err
	case *model.Group:
		_, err := tx.ExecContext(ctx, `INSERT INTO groups VALUES(?,?,?,?,?,?,?,?)`, sourceID, v.ID, v.CaseID, v.Name, record.Line, byteOffset, byteLength, []byte(record.Raw))
		return err
	case *model.Trajectory:
		_, err := tx.ExecContext(ctx, `INSERT INTO trajectories VALUES(?,?,?,?,?,?,?,?,?,?,?)`, sourceID, v.ID, v.GroupID,
			v.ParentID, v.BranchID, v.Status, v.Termination, record.Line, byteOffset, byteLength, []byte(record.Raw))
		return err
	case *model.Event:
		var sourcePath any
		var sourceLine, offset, length any
		if v.Source != nil {
			sourcePath, sourceLine, offset, length = v.Source.Path, ptrValue(v.Source.Line), ptrValue(v.Source.ByteOffset), ptrValue(v.Source.ByteLength)
		} else if source.Adapter == "" {
			sourcePath, sourceLine, offset, length = source.Path, record.Line, byteOffset, byteLength
		}
		var contextPresent int
		var contextOperation, contextInputTokens, contextInputTokensBefore, contextCapacity, contextProvenance any
		if v.Context != nil {
			contextPresent = 1
			contextOperation = nullableString(v.Context.Operation)
			contextInputTokens = ptrValue(v.Context.InputTokens)
			contextInputTokensBefore = ptrValue(v.Context.InputTokensBefore)
			contextCapacity = ptrValue(v.Context.Capacity)
			contextProvenance = nullableString(v.Context.Provenance)
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO events(
			source_id,id,trajectory_id,sequence,kind,timestamp,parent_id,branch_id,alignment_key,state_hash,
			search_text,source_path,source_line,byte_offset,byte_length,line,record_byte_offset,record_byte_length,raw,
			context_present,context_operation,context_input_tokens,context_input_tokens_before,context_capacity,context_provenance
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, sourceID, v.ID,
			v.TrajectoryID, v.Sequence, v.Kind, v.Timestamp, v.ParentID, v.BranchID, v.AlignmentKey, v.StateHash,
			eventSearchText(v), sourcePath, sourceLine, offset, length, record.Line, byteOffset, byteLength, []byte(record.Raw),
			contextPresent, contextOperation, contextInputTokens, contextInputTokensBefore, contextCapacity, contextProvenance)
		return err
	case *model.Signal:
		_, err := tx.ExecContext(ctx, `INSERT INTO signals VALUES(?,?,?,?,?,?,?,?,?,?)`, sourceID, v.ID, v.TrajectoryID,
			v.EventID, v.Name, v.Unit, record.Line, byteOffset, byteLength, []byte(record.Raw))
		return err
	case *model.Artifact:
		_, err := tx.ExecContext(ctx, `INSERT INTO artifacts VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, sourceID, v.ID, v.TrajectoryID,
			v.EventID, v.Name, v.MediaType, v.Path, v.SHA256, record.Line, byteOffset, byteLength, []byte(record.Raw))
		return err
	case *model.Complete:
		return nil
	default:
		return fmt.Errorf("unsupported record value %T", v)
	}
}

func eventSearchText(event *model.Event) string {
	// Search the human-meaningful payload once. Excluding event.Raw avoids a
	// second full copy of source-native payloads (which canonical raw already
	// preserves) while retaining titles, tool names, arguments, output, and
	// adapter metadata.
	value := struct {
		Kind         string         `json:"kind"`
		AlignmentKey string         `json:"alignment_key,omitempty"`
		Input        any            `json:"input,omitempty"`
		Output       any            `json:"output,omitempty"`
		Data         any            `json:"data,omitempty"`
		Metadata     model.Metadata `json:"metadata,omitempty"`
	}{event.Kind, event.AlignmentKey, event.Input, event.Output, event.Data, event.Metadata}
	encoded, err := json.Marshal(value)
	if err != nil {
		return event.Kind + " " + event.AlignmentKey
	}
	return string(encoded)
}

func ptrValue(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func recordID(value any) string {
	switch v := value.(type) {
	case *model.Run:
		return v.ID
	case *model.Case:
		return v.ID
	case *model.Group:
		return v.ID
	case *model.Trajectory:
		return v.ID
	case *model.Event:
		return v.ID
	case *model.Signal:
		return v.ID
	case *model.Artifact:
		return v.ID
	default:
		return ""
	}
}
