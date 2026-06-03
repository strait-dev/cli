package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// printDataState builds an appState that writes machine output to buf.
func printDataState(buf *bytes.Buffer, opts *rootOptions) *appState {
	return &appState{opts: opts, stdout: buf}
}

// TestPrintData_RawMessageRendersAllFormats verifies that a json.RawMessage
// payload (as returned by the coverage commands) is decoded so non-JSON
// renderers work, instead of being treated as raw bytes.
func TestPrintData_RawMessageRendersAllFormats(t *testing.T) {
	t.Parallel()

	list := json.RawMessage(`[{"id":"a","name":"alpha"},{"id":"b","name":"beta"}]`)

	t.Run("yaml list", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		if err := printData(printDataState(&buf, &rootOptions{outputFormat: "yaml"}), list); err != nil {
			t.Fatalf("printData: %v", err)
		}
		out := buf.String()
		// Raw-byte mishandling would emit a base64 string or a list of numbers;
		// correct decoding yields readable YAML keys.
		if !strings.Contains(out, "name: alpha") || !strings.Contains(out, "id: b") {
			t.Fatalf("expected decoded yaml, got: %q", out)
		}
	})

	t.Run("jsonpath into list", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		opts := &rootOptions{outputFormat: "jsonpath", outputPath: "$[0].name"}
		if err := printData(printDataState(&buf, opts), list); err != nil {
			t.Fatalf("printData: %v", err)
		}
		if !strings.Contains(buf.String(), "alpha") {
			t.Fatalf("expected jsonpath to extract alpha, got: %q", buf.String())
		}
	})

	t.Run("quiet prints ids", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		if err := printData(printDataState(&buf, &rootOptions{quiet: true}), list); err != nil {
			t.Fatalf("printData: %v", err)
		}
		got := strings.Fields(buf.String())
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("expected ids [a b], got: %v", got)
		}
	})

	t.Run("single object quiet", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		obj := json.RawMessage(`{"id":"solo","name":"x"}`)
		if err := printData(printDataState(&buf, &rootOptions{quiet: true}), obj); err != nil {
			t.Fatalf("printData: %v", err)
		}
		if strings.TrimSpace(buf.String()) != "solo" {
			t.Fatalf("expected id solo, got: %q", buf.String())
		}
	})

	t.Run("empty payload renders without error", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		if err := printData(printDataState(&buf, &rootOptions{outputFormat: "yaml"}), json.RawMessage("")); err != nil {
			t.Fatalf("printData on empty raw: %v", err)
		}
	})
}
