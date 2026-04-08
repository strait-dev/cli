package main

import (
	"errors"
	"testing"
)

func TestExitCodeFromError_NilIsOK(t *testing.T) {
	t.Parallel()
	if got := exitCodeFromError(nil); got != ExitOK {
		t.Fatalf("expected ExitOK, got %d", got)
	}
}

func TestExitCodeFromError_HTTPStatusMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		msg  string
		want int
	}{
		{"request failed (400): bad field", ExitValidation},
		{"request failed (422): unprocessable", ExitValidation},
		{"request failed (401): unauthorized", ExitAuthError},
		{"request failed (403): forbidden", ExitAuthError},
		{"request failed (404): not found", ExitNotFound},
		{"request failed (409): conflict", ExitConflict},
		{"request failed (500): internal server error", ExitServerError},
		{"request failed (503): service unavailable", ExitServerError},
	}

	for _, tc := range cases {
		got := exitCodeFromError(errors.New(tc.msg))
		if got != tc.want {
			t.Errorf("exitCodeFromError(%q) = %d, want %d", tc.msg, got, tc.want)
		}
	}
}

func TestExitCodeFromError_ConfigErrors(t *testing.T) {
	t.Parallel()

	cases := []string{
		"project ID is required",
		"--job is required",
		"API key not set",
		"server URL is missing",
		"non-interactive mode: use --yes",
		"interactive prompt blocked in non-interactive mode",
	}

	for _, msg := range cases {
		got := exitCodeFromError(errors.New(msg))
		if got != ExitConfigError {
			t.Errorf("exitCodeFromError(%q) = %d, want ExitConfigError (%d)", msg, got, ExitConfigError)
		}
	}
}

func TestExitCodeFromError_UnknownIsGeneral(t *testing.T) {
	t.Parallel()

	got := exitCodeFromError(errors.New("something completely unexpected"))
	if got != ExitGeneralError {
		t.Fatalf("expected ExitGeneralError, got %d", got)
	}
}

func TestExitCodeConstants_AreDistinct(t *testing.T) {
	t.Parallel()

	codes := []int{ExitOK, ExitGeneralError, ExitPanic, ExitConfigError, ExitAuthError, ExitNotFound, ExitConflict, ExitValidation, ExitServerError}
	seen := make(map[int]bool)
	for _, c := range codes {
		if seen[c] {
			t.Fatalf("duplicate exit code: %d", c)
		}
		seen[c] = true
	}
}
