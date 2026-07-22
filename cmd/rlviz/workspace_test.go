package main

import "testing"

func TestFailedTrajectoryUsesVerdictThenTerminalState(t *testing.T) {
	passed, failed := true, false
	for _, test := range []struct {
		name        string
		success     *bool
		status      string
		termination string
		want        bool
	}{
		{name: "explicit pass wins", success: &passed, status: "failed", want: false},
		{name: "explicit failure", success: &failed, status: "completed", want: true},
		{name: "failed status without verdict", status: "failed", want: true},
		{name: "infrastructure termination", termination: "infrastructure_error", want: true},
		{name: "policy termination", termination: "policy_violation", want: true},
		{name: "completed without verdict", status: "completed", termination: "success", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := failedTrajectory(test.success, test.status, test.termination); got != test.want {
				t.Fatalf("failedTrajectory() = %v, want %v", got, test.want)
			}
		})
	}
}
