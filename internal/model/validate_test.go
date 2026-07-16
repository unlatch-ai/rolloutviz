package model

import (
	"strings"
	"testing"
)

func TestRelationshipValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		stream string
		want   string
	}{
		{
			name: "forward run reference",
			stream: lines(
				`{"record_type":"case","id":"case","run_id":"later"}`,
				`{"record_type":"run","id":"later"}`,
				`{"record_type":"complete","records":2,"warnings":0}`,
			),
			want: "unknown or later run",
		},
		{
			name: "cross trajectory event target",
			stream: lines(
				`{"record_type":"run","id":"run"}`,
				`{"record_type":"case","id":"case","run_id":"run"}`,
				`{"record_type":"group","id":"group","case_id":"case"}`,
				`{"record_type":"trajectory","id":"a","group_id":"group"}`,
				`{"record_type":"trajectory","id":"b","group_id":"group"}`,
				`{"record_type":"event","id":"event-a","trajectory_id":"a","sequence":0,"kind":"log"}`,
				`{"record_type":"signal","id":"signal-b","trajectory_id":"b","event_id":"event-a","name":"reward","value":0}`,
				`{"record_type":"complete","records":7,"warnings":0}`,
			),
			want: "belongs to another trajectory",
		},
		{
			name: "incorrect completion count",
			stream: lines(
				`{"record_type":"run","id":"run"}`,
				`{"record_type":"complete","records":2,"warnings":0}`,
			),
			want: "reports 2 records, decoded 1",
		},
		{
			name: "ambiguous artifact location",
			stream: lines(
				`{"record_type":"run","id":"run"}`,
				`{"record_type":"case","id":"case","run_id":"run"}`,
				`{"record_type":"group","id":"group","case_id":"case"}`,
				`{"record_type":"trajectory","id":"traj","group_id":"group"}`,
				`{"record_type":"artifact","id":"artifact","trajectory_id":"traj","media_type":"text/plain","path":"a","text":"b"}`,
				`{"record_type":"complete","records":5,"warnings":0}`,
			),
			want: "exactly one of path, text, or json",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := Decode(strings.NewReader(tt.stream), nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func lines(values ...string) string { return strings.Join(values, "\n") + "\n" }
