package output

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderJSONL_SliceEmitsOneLinePerItem(t *testing.T) {
	t.Parallel()

	data := []map[string]any{
		{"id": "run_1", "status": "completed"},
		{"id": "run_2", "status": "failed"},
		{"id": "run_3", "status": "queued"},
	}
	out, err := RenderToString(data, Options{Format: "jsonl"})
	if err != nil {
		t.Fatalf("render jsonl: %v", err)
	}

	lines := nonEmptyLines(out)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), out)
	}
}

func TestRenderJSONL_EachLineIsValidJSON(t *testing.T) {
	t.Parallel()

	data := []map[string]any{
		{"id": "a", "val": 1},
		{"id": "b", "val": 2},
	}
	out, err := RenderToString(data, Options{Format: "jsonl"})
	if err != nil {
		t.Fatalf("render jsonl: %v", err)
	}

	for _, line := range nonEmptyLines(out) {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("line is not valid JSON: %q — %v", line, err)
		}
	}
}

func TestRenderJSONL_SingleObjectEmitsOneLine(t *testing.T) {
	t.Parallel()

	data := map[string]any{"id": "job_1", "status": "active"}
	out, err := RenderToString(data, Options{Format: "jsonl"})
	if err != nil {
		t.Fatalf("render jsonl: %v", err)
	}

	lines := nonEmptyLines(out)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line for single object, got %d:\n%s", len(lines), out)
	}
}

func TestRenderJSONL_NoJSONArray(t *testing.T) {
	t.Parallel()

	data := []map[string]any{{"id": "x"}, {"id": "y"}}
	out, err := RenderToString(data, Options{Format: "jsonl"})
	if err != nil {
		t.Fatalf("render jsonl: %v", err)
	}

	// Must not be wrapped in a JSON array.
	trimmed := strings.TrimSpace(out)
	if strings.HasPrefix(trimmed, "[") || strings.HasSuffix(trimmed, "]") {
		t.Errorf("JSONL output must not be a JSON array, got: %s", out)
	}
}

func TestRenderJSONL_EmptySliceEmitsNothing(t *testing.T) {
	t.Parallel()

	data := []map[string]any{}
	out, err := RenderToString(data, Options{Format: "jsonl"})
	if err != nil {
		t.Fatalf("render jsonl: %v", err)
	}

	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty output for empty slice, got: %q", out)
	}
}

func TestRenderJSONL_PreservesTypes(t *testing.T) {
	t.Parallel()

	data := []map[string]any{
		{"id": "run_1", "attempt": 3, "ok": true, "val": 1.5},
	}
	out, err := RenderToString(data, Options{Format: "jsonl"})
	if err != nil {
		t.Fatalf("render jsonl: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if attempt, ok := m["attempt"].(float64); !ok || attempt != 3 {
		t.Errorf("expected attempt=3 (float64), got %T(%v)", m["attempt"], m["attempt"])
	}
	if ok, _ := m["ok"].(bool); !ok {
		t.Errorf("expected ok=true (bool), got %T(%v)", m["ok"], m["ok"])
	}
}

// nonEmptyLines returns lines in s that are not empty after trimming.
func nonEmptyLines(s string) []string {
	var out []string
	for line := range strings.SplitSeq(s, "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
