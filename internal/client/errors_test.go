package client

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *APIError
		want string
	}{
		{name: "with op", err: &APIError{StatusCode: 404, Message: "not found", Op: "request"}, want: "request failed (404): not found"},
		{name: "default op", err: &APIError{StatusCode: 500, Message: "boom"}, want: "request failed (500): boom"},
		{name: "upload op", err: &APIError{StatusCode: 403, Message: "denied", Op: "upload"}, want: "upload failed (403): denied"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.err.Error(); got != tc.want {
				t.Fatalf("Error: got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAPIError_AsAndIs(t *testing.T) {
	t.Parallel()

	base := &APIError{StatusCode: 404, Message: "not found", Op: "request"}
	wrapped := fmt.Errorf("resolving job %q: %w", "j-1", base)

	var apiErr *APIError
	if !errors.As(wrapped, &apiErr) {
		t.Fatal("errors.As failed to extract APIError from wrapped error")
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode: got %d, want 404", apiErr.StatusCode)
	}
}

func TestIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "404 directly", err: &APIError{StatusCode: 404}, want: true},
		{name: "404 wrapped", err: fmt.Errorf("ctx: %w", &APIError{StatusCode: 404}), want: true},
		{name: "500", err: &APIError{StatusCode: 500}, want: false},
		{name: "403", err: &APIError{StatusCode: 403}, want: false},
		{name: "transport error", err: errors.New("dial tcp: refused"), want: false},
		{name: "nil", err: nil, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsNotFound(tc.err); got != tc.want {
				t.Fatalf("IsNotFound(%v): got %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestIsAPIErrorWithStatus(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("ctx: %w", &APIError{StatusCode: 503})
	if !IsAPIErrorWithStatus(err, 503) {
		t.Fatal("expected match for 503")
	}
	if IsAPIErrorWithStatus(err, 500) {
		t.Fatal("did not expect match for 500")
	}
	if IsAPIErrorWithStatus(errors.New("plain"), 503) {
		t.Fatal("did not expect match for non-APIError")
	}
}
