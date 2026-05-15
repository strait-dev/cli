package types

import (
	"testing"
)

func TestRunStatus_IsTerminal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   RunStatus
		terminal bool
	}{
		{StatusDelayed, false},
		{StatusQueued, false},
		{StatusDequeued, false},
		{StatusExecuting, false},
		{StatusWaiting, false},
		{StatusPaused, false},
		{StatusReplayStaged, false},
		{StatusDeadLetter, false},
		{StatusCompleted, true},
		{StatusFailed, true},
		{StatusTimedOut, true},
		{StatusCrashed, true},
		{StatusSystemFailed, true},
		{StatusCanceled, true},
		{StatusExpired, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			if got := tt.status.IsTerminal(); got != tt.terminal {
				t.Errorf("RunStatus(%q).IsTerminal() = %v, want %v", tt.status, got, tt.terminal)
			}
		})
	}
}

func TestRunStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status RunStatus
		valid  bool
	}{
		{StatusCompleted, true},
		{StatusFailed, true},
		{StatusExecuting, true},
		{StatusDeadLetter, true},
		{StatusReplayStaged, true},
		{StatusPaused, true},
		{"unknown", false},
		{"", false},
		{"COMPLETED", false},
	}

	for _, tt := range tests {
		name := string(tt.status)
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("RunStatus(%q).IsValid() = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}

func TestTerminalStatuses(t *testing.T) {
	t.Parallel()

	statuses := TerminalStatuses()
	if len(statuses) == 0 {
		t.Fatal("TerminalStatuses() returned empty slice")
	}

	for _, s := range statuses {
		if !s.IsTerminal() {
			t.Errorf("TerminalStatuses() contains non-terminal status %q", s)
		}
	}
}

func TestWorkflowRunStatus_IsTerminal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   WorkflowRunStatus
		terminal bool
	}{
		{WfStatusPending, false},
		{WfStatusRunning, false},
		{WfStatusPaused, false},
		{WfStatusCompleted, true},
		{WfStatusFailed, true},
		{WfStatusTimedOut, true},
		{WfStatusCanceled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			if got := tt.status.IsTerminal(); got != tt.terminal {
				t.Errorf("WorkflowRunStatus(%q).IsTerminal() = %v, want %v", tt.status, got, tt.terminal)
			}
		})
	}
}

func TestWorkflowRunStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status WorkflowRunStatus
		valid  bool
	}{
		{WfStatusPending, true},
		{WfStatusRunning, true},
		{WfStatusCompleted, true},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		name := string(tt.status)
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("WorkflowRunStatus(%q).IsValid() = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}

func TestStepRunStatus_IsTerminal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   StepRunStatus
		terminal bool
	}{
		{StepPending, false},
		{StepWaiting, false},
		{StepRunning, false},
		{StepCompleted, true},
		{StepFailed, true},
		{StepSkipped, true},
		{StepCanceled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			if got := tt.status.IsTerminal(); got != tt.terminal {
				t.Errorf("StepRunStatus(%q).IsTerminal() = %v, want %v", tt.status, got, tt.terminal)
			}
		})
	}
}

func TestVersionPolicy_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		policy VersionPolicy
		valid  bool
	}{
		{VersionPolicyPin, true},
		{VersionPolicyLatest, true},
		{VersionPolicyMinor, true},
		{"major", false},
		{"", false},
	}

	for _, tt := range tests {
		name := string(tt.policy)
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tt.policy.IsValid(); got != tt.valid {
				t.Errorf("VersionPolicy(%q).IsValid() = %v, want %v", tt.policy, got, tt.valid)
			}
		})
	}
}

func TestExecutionMode_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode  ExecutionMode
		valid bool
	}{
		{ExecutionModeHTTP, true},
		{"managed", false},
		{"serverless", false},
		{"", false},
	}

	for _, tt := range tests {
		name := string(tt.mode)
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tt.mode.IsValid(); got != tt.valid {
				t.Errorf("ExecutionMode(%q).IsValid() = %v, want %v", tt.mode, got, tt.valid)
			}
		})
	}
}
