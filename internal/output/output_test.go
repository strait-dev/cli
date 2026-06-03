package output

import (
	"strings"
	"testing"
)

func TestRenderJSON(t *testing.T) {
	t.Parallel()

	out, err := RenderToString([]map[string]any{{"id": "run_1", "status": "completed"}}, Options{Format: "json"})
	if err != nil {
		t.Fatalf("render json: %v", err)
	}
	if !strings.Contains(out, "run_1") {
		t.Fatalf("unexpected json output: %s", out)
	}
}

func TestRenderTable(t *testing.T) {
	t.Parallel()

	out, err := RenderToString([]map[string]any{{"id": "job_1", "status": "queued"}}, Options{Format: "table", TTY: true})
	if err != nil {
		t.Fatalf("render table: %v", err)
	}
	if !strings.Contains(out, "id") || !strings.Contains(out, "job_1") {
		t.Fatalf("unexpected table output: %s", out)
	}
}

func TestRenderCSV(t *testing.T) {
	t.Parallel()

	out, err := RenderToString([]map[string]any{{"id": "job_1", "status": "queued"}}, Options{Format: "csv"})
	if err != nil {
		t.Fatalf("render csv: %v", err)
	}
	if !strings.Contains(out, "id,status") {
		t.Fatalf("unexpected csv output: %s", out)
	}
}

func TestRenderGoTemplate(t *testing.T) {
	t.Parallel()

	out, err := RenderToString(map[string]any{"id": "run_9"}, Options{Format: "go-template", Template: "{{.id}}"})
	if err != nil {
		t.Fatalf("render go-template: %v", err)
	}
	if strings.TrimSpace(out) != "run_9" {
		t.Fatalf("unexpected template output: %s", out)
	}
}

func TestRenderJSONPath(t *testing.T) {
	t.Parallel()

	out, err := RenderToString(map[string]any{"data": map[string]any{"id": "job_123"}}, Options{Format: "jsonpath", JSONPath: "$.data.id"})
	if err != nil {
		t.Fatalf("render jsonpath: %v", err)
	}
	if strings.TrimSpace(out) != "job_123" {
		t.Fatalf("unexpected jsonpath output: %s", out)
	}
}

func TestRenderYAML_SingleObject(t *testing.T) {
	t.Parallel()

	out, err := RenderToString(map[string]any{"id": "job_1", "status": "queued"}, Options{Format: "yaml"})
	if err != nil {
		t.Fatalf("render yaml: %v", err)
	}
	if !strings.Contains(out, "id: job_1") {
		t.Fatalf("unexpected yaml output: %s", out)
	}
	if !strings.Contains(out, "status: queued") {
		t.Fatalf("unexpected yaml output: %s", out)
	}
}

func TestRenderYAML_Slice(t *testing.T) {
	t.Parallel()

	data := []map[string]any{
		{"id": "job_1", "name": "first"},
		{"id": "job_2", "name": "second"},
	}
	out, err := RenderToString(data, Options{Format: "yaml"})
	if err != nil {
		t.Fatalf("render yaml slice: %v", err)
	}
	if !strings.Contains(out, "job_1") || !strings.Contains(out, "job_2") {
		t.Fatalf("unexpected yaml slice output: %s", out)
	}
}

func TestRenderYAML_Empty(t *testing.T) {
	t.Parallel()

	out, err := RenderToString([]map[string]any{}, Options{Format: "yaml"})
	if err != nil {
		t.Fatalf("render empty yaml: %v", err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("expected empty yaml array, got: %s", out)
	}
}

func TestRenderTable_NoHeaders(t *testing.T) {
	t.Parallel()

	out, err := RenderToString([]map[string]any{{"id": "job_1"}}, Options{Format: "table", TTY: true, NoHeaders: true})
	if err != nil {
		t.Fatalf("render no-headers: %v", err)
	}
	if strings.Contains(out, "ID") {
		t.Fatalf("expected no header row, got: %s", out)
	}
	if !strings.Contains(out, "job_1") {
		t.Fatalf("expected data row, got: %s", out)
	}
}

func TestRenderTable_EmptySlice(t *testing.T) {
	t.Parallel()

	out, err := RenderToString([]map[string]any{}, Options{Format: "table", TTY: true})
	if err != nil {
		t.Fatalf("render empty table: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty output for empty slice, got: %q", out)
	}
}

func TestRenderUnsupportedFormat(t *testing.T) {
	t.Parallel()

	_, err := RenderToString(map[string]any{"id": "1"}, Options{Format: "xml"})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported format error, got: %v", err)
	}
}

func TestRenderGoTemplate_EmptyTemplate(t *testing.T) {
	t.Parallel()

	_, err := RenderToString(map[string]any{"id": "1"}, Options{Format: "go-template", Template: ""})
	if err == nil {
		t.Fatal("expected error for empty template")
	}
}

func TestRenderJSONPath_EmptyExpr(t *testing.T) {
	t.Parallel()

	_, err := RenderToString(map[string]any{"id": "1"}, Options{Format: "jsonpath", JSONPath: ""})
	if err == nil {
		t.Fatal("expected error for empty jsonpath")
	}
}

func TestRenderJSON_SingleObject(t *testing.T) {
	t.Parallel()

	out, err := RenderToString(map[string]any{"key": "value"}, Options{Format: "json"})
	if err != nil {
		t.Fatalf("render json: %v", err)
	}
	if !strings.Contains(out, `"key": "value"`) {
		t.Fatalf("unexpected json output: %s", out)
	}
}

func TestRenderCSV_MultipleRows(t *testing.T) {
	t.Parallel()

	data := []map[string]any{
		{"id": "1", "name": "first"},
		{"id": "2", "name": "second"},
	}
	out, err := RenderToString(data, Options{Format: "csv"})
	if err != nil {
		t.Fatalf("render csv: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 { // header + 2 data rows
		t.Fatalf("expected 3 lines (header + 2 rows), got %d: %s", len(lines), out)
	}
}

// TestRenderTable_AnySliceOfMaps verifies that a []any whose elements are maps
// (as produced by decoding a generic JSON payload) renders as a table rather
// than erroring on the interface-kind elements.
func TestRenderTable_AnySliceOfMaps(t *testing.T) {
	t.Parallel()

	data := []any{
		map[string]any{"id": "a", "name": "alpha"},
		map[string]any{"id": "b", "name": "beta"},
	}
	out, err := RenderToString(data, Options{Format: "table"})
	if err != nil {
		t.Fatalf("render table: %v", err)
	}
	for _, want := range []string{"id", "name", "alpha", "beta"} {
		if !strings.Contains(out, want) {
			t.Fatalf("table missing %q, got:\n%s", want, out)
		}
	}
}

// TestRenderTable_Columns verifies that Options.Columns restricts and orders
// table columns, with unknown columns rendering as empty cells.
func TestRenderTable_Columns(t *testing.T) {
	t.Parallel()

	data := []any{
		map[string]any{"id": "a", "name": "alpha", "secret": "x", "created_at": "t1"},
		map[string]any{"id": "b", "name": "beta", "secret": "y", "created_at": "t2"},
	}
	out, err := RenderToString(data, Options{Format: "table", Columns: []string{"id", "name", "missing"}})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// Selected columns and their values appear; unselected ones do not.
	for _, want := range []string{"id", "name", "missing", "alpha", "beta"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output:\n%s", want, out)
		}
	}
	for _, notWant := range []string{"secret", "created_at", "x", "y", "t1"} {
		if strings.Contains(out, notWant) {
			t.Fatalf("did not expect %q in projected output:\n%s", notWant, out)
		}
	}
	// Header order is preserved.
	headerLine := strings.SplitN(out, "\n", 2)[0]
	if strings.Index(headerLine, "id") > strings.Index(headerLine, "name") {
		t.Fatalf("expected id before name in header: %q", headerLine)
	}
}
