package output

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderCompact_SingleObjectStripsZeros(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"id":     "job_1",
		"name":   "my-job",
		"count":  0,
		"active": false,
		"note":   "",
		"tags":   []any{},
	}
	out, err := RenderToString(data, Options{Format: "compact"})
	if err != nil {
		t.Fatalf("render compact: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	if _, ok := m["count"]; ok {
		t.Error("expected count=0 to be stripped")
	}
	if _, ok := m["active"]; ok {
		t.Error("expected active=false to be stripped")
	}
	if _, ok := m["note"]; ok {
		t.Error("expected note=\"\" to be stripped")
	}
	if _, ok := m["tags"]; ok {
		t.Error("expected tags=[] to be stripped")
	}

	// Non-zero fields must remain.
	if m["id"] != "job_1" {
		t.Errorf("expected id=job_1, got %v", m["id"])
	}
	if m["name"] != "my-job" {
		t.Errorf("expected name=my-job, got %v", m["name"])
	}
}

func TestRenderCompact_SliceStripsZerosPerElement(t *testing.T) {
	t.Parallel()

	data := []map[string]any{
		{"id": "r1", "status": "completed", "error": ""},
		{"id": "r2", "status": "failed", "error": "timeout"},
	}
	out, err := RenderToString(data, Options{Format: "compact"})
	if err != nil {
		t.Fatalf("render compact: %v", err)
	}

	var items []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &items); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if _, ok := items[0]["error"]; ok {
		t.Error("expected empty error field stripped from first item")
	}
	if items[1]["error"] != "timeout" {
		t.Errorf("expected error=timeout in second item, got %v", items[1]["error"])
	}
}

func TestRenderCompact_OutputIsSingleLine(t *testing.T) {
	t.Parallel()

	data := map[string]any{"id": "x", "val": 42}
	out, err := RenderToString(data, Options{Format: "compact"})
	if err != nil {
		t.Fatalf("render compact: %v", err)
	}

	lines := nonEmptyLines(out)
	if len(lines) != 1 {
		t.Fatalf("expected single-line output, got %d lines:\n%s", len(lines), out)
	}
}

func TestRenderCompact_NestedObjectStripped(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"id": "job_1",
		"meta": map[string]any{
			"region": "us-east-1",
			"weight": 0,
			"label":  "",
		},
	}
	out, err := RenderToString(data, Options{Format: "compact"})
	if err != nil {
		t.Fatalf("render compact: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	meta, ok := m["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta to be a map, got %T", m["meta"])
	}
	if _, ok := meta["weight"]; ok {
		t.Error("expected nested weight=0 to be stripped")
	}
	if _, ok := meta["label"]; ok {
		t.Error("expected nested label=\"\" to be stripped")
	}
	if meta["region"] != "us-east-1" {
		t.Errorf("expected region=us-east-1, got %v", meta["region"])
	}
}

func TestRenderCompact_PreservesNonZeroBool(t *testing.T) {
	t.Parallel()

	data := map[string]any{"id": "j1", "active": true, "paused": false}
	out, err := RenderToString(data, Options{Format: "compact"})
	if err != nil {
		t.Fatalf("render compact: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	if v, ok := m["active"].(bool); !ok || !v {
		t.Errorf("expected active=true to be preserved, got %v", m["active"])
	}
	if _, ok := m["paused"]; ok {
		t.Error("expected paused=false to be stripped")
	}
}

func TestStripZeroValues_NilInput(t *testing.T) {
	t.Parallel()

	result := stripZeroValues(nil)
	if result != nil {
		t.Errorf("expected nil output for nil input, got %v", result)
	}
}

func TestRenderCompact_EmptySliceInOutput(t *testing.T) {
	t.Parallel()

	data := []map[string]any{}
	out, err := RenderToString(data, Options{Format: "compact"})
	if err != nil {
		t.Fatalf("render compact: %v", err)
	}

	trimmed := strings.TrimSpace(out)
	if trimmed != "[]" {
		t.Errorf("expected [] for empty slice, got %q", trimmed)
	}
}
