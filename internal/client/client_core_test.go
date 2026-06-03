package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseErrorBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		wantMessage string
		wantCode    string
	}{
		{
			name:        "object envelope",
			body:        `{"error":{"code":"bad_request","message":"endpoint_url is required"}}`,
			wantMessage: "endpoint_url is required",
			wantCode:    "bad_request",
		},
		{
			name:        "object envelope code only",
			body:        `{"error":{"code":"forbidden"}}`,
			wantMessage: `{"error":{"code":"forbidden"}}`,
			wantCode:    "forbidden",
		},
		{
			name:        "string envelope",
			body:        `{"error":"something broke"}`,
			wantMessage: "something broke",
			wantCode:    "",
		},
		{
			name:        "plain non-json body",
			body:        "404 page not found",
			wantMessage: "404 page not found",
			wantCode:    "",
		},
		{
			name:        "empty body",
			body:        "",
			wantMessage: "",
			wantCode:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			msg, code := parseErrorBody([]byte(tc.body))
			if msg != tc.wantMessage {
				t.Errorf("message = %q, want %q", msg, tc.wantMessage)
			}
			if code != tc.wantCode {
				t.Errorf("code = %q, want %q", code, tc.wantCode)
			}
		})
	}
}

func TestUnmarshalListBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		wantLen int
		wantErr bool
	}{
		{name: "bare array", body: `[{"id":"a"},{"id":"b"}]`, wantLen: 2},
		{name: "paginated envelope", body: `{"data":[{"id":"a"}],"has_more":false}`, wantLen: 1},
		{name: "empty envelope data", body: `{"data":[],"has_more":false}`, wantLen: 0},
		{name: "absent data field", body: `{"has_more":false}`, wantLen: 0},
		{name: "empty bare array", body: `[]`, wantLen: 0},
		{name: "malformed", body: `{`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out []map[string]any
			err := unmarshalListBody([]byte(tc.body), &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(out) != tc.wantLen {
				t.Errorf("len = %d, want %d", len(out), tc.wantLen)
			}
		})
	}
}

// TestDoJSONNoContent verifies that a 204 response with a non-nil out pointer
// does not return io.EOF (the regression that broke every delete/update).
func TestDoJSONNoContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, err := New(srv.URL, "key", 10*time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var out map[string]any
	if err := c.doJSON(context.Background(), http.MethodDelete, "/v1/things/abc", nil, nil, &out); err != nil {
		t.Fatalf("doJSON on 204 returned error: %v", err)
	}
}

// TestDoListJSONAcceptsBothShapes verifies list decoding works against both a
// bare-array endpoint and a paginated-envelope endpoint.
func TestDoListJSONAcceptsBothShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "bare array", body: `[{"id":"a"},{"id":"b"}]`},
		{name: "envelope", body: `{"data":[{"id":"a"},{"id":"b"}],"has_more":false}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			c, err := New(srv.URL, "key", 10*time.Second)
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			var out []map[string]any
			if err := c.doListJSON(context.Background(), "/v1/things", nil, &out); err != nil {
				t.Fatalf("doListJSON: %v", err)
			}
			if len(out) != 2 {
				t.Errorf("len = %d, want 2", len(out))
			}
		})
	}
}

// TestDoJSONErrorEnvelope verifies the structured error code/message surface
// through APIError.
func TestDoJSONErrorEnvelope(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"validation_error","message":"value is required"}}`))
	}))
	defer srv.Close()

	c, err := New(srv.URL, "key", 10*time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var out map[string]any
	err = c.doJSON(context.Background(), http.MethodPost, "/v1/things", nil, map[string]string{}, &out)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != "validation_error" {
		t.Errorf("Code = %q, want validation_error", apiErr.Code)
	}
	if apiErr.Message != "value is required" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "value is required")
	}
}
