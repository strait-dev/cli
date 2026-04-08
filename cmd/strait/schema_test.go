package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestSchemaCommand_HasSubcommands(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newSchemaCommand(state)

	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}

	for _, want := range []string{"runtimes", "job", "deployment", "workflow", "run", "trigger", "secret", "api-key"} {
		if !names[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

func TestSchemaRuntimes_OutputIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newSchemaCommand(state)
	cmd.SetArgs([]string{"runtimes"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var runtimes []map[string]string
	if err := json.Unmarshal([]byte(out), &runtimes); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if len(runtimes) == 0 {
		t.Fatal("expected non-empty runtimes list")
	}
	// Check known runtimes are present
	names := make(map[string]bool)
	for _, r := range runtimes {
		names[r["runtime"]] = true
	}
	for _, want := range []string{"go", "python", "typescript", "ruby", "rust"} {
		if !names[want] {
			t.Errorf("missing runtime %q in output", want)
		}
	}
}

func TestSchemaJob_OutputIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newSchemaCommand(state)
	cmd.SetArgs([]string{"job"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var schema schemaResource
	if err := json.Unmarshal([]byte(out), &schema); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if schema.Resource != "job" {
		t.Errorf("expected resource=job, got %q", schema.Resource)
	}
	if len(schema.Fields) == 0 {
		t.Fatal("expected non-empty fields")
	}
	// Check required fields exist
	fieldNames := make(map[string]bool)
	for _, f := range schema.Fields {
		fieldNames[f.Name] = true
	}
	for _, want := range []string{"id", "name", "slug", "project_id"} {
		if !fieldNames[want] {
			t.Errorf("missing field %q in job schema", want)
		}
	}
}

func TestSchemaDeployment_OutputIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newSchemaCommand(state)
	cmd.SetArgs([]string{"deployment"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var schema schemaResource
	if err := json.Unmarshal([]byte(out), &schema); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if schema.Resource != "deployment" {
		t.Errorf("expected resource=deployment, got %q", schema.Resource)
	}
	// Verify status enum includes expected values
	for _, f := range schema.Fields {
		if f.Name == "status" {
			found := false
			for _, e := range f.Enum {
				if e == "ready" {
					found = true
				}
			}
			if !found {
				t.Errorf("status enum missing 'ready', got: %v", f.Enum)
			}
			return
		}
	}
	t.Error("missing status field in deployment schema")
}

func TestSchemaWorkflow_OutputIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newSchemaCommand(state)
	cmd.SetArgs([]string{"workflow"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var schema schemaResource
	if err := json.Unmarshal([]byte(out), &schema); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if schema.Resource != "workflow" {
		t.Errorf("expected resource=workflow, got %q", schema.Resource)
	}
}

func TestSchemaRun_ContainsRunStatuses(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newSchemaCommand(state)
	cmd.SetArgs([]string{"run"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// All known run statuses should appear in the output.
	for _, status := range []string{"queued", "executing", "completed", "failed", "timed_out", "canceled"} {
		if !strings.Contains(out, status) {
			t.Errorf("expected run status %q in schema output", status)
		}
	}
}

func TestSchemaTrigger_OutputIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newSchemaCommand(state)
	cmd.SetArgs([]string{"trigger"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var schema schemaResource
	if err := json.Unmarshal([]byte(out), &schema); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if schema.Resource != "trigger" {
		t.Errorf("expected resource=trigger, got %q", schema.Resource)
	}
	fieldNames := make(map[string]bool)
	for _, f := range schema.Fields {
		fieldNames[f.Name] = true
	}
	for _, want := range []string{"event", "target_type", "target_id", "enabled"} {
		if !fieldNames[want] {
			t.Errorf("missing field %q in trigger schema", want)
		}
	}
}

func TestSchemaSecret_OutputIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newSchemaCommand(state)
	cmd.SetArgs([]string{"secret"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var schema schemaResource
	if err := json.Unmarshal([]byte(out), &schema); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if schema.Resource != "secret" {
		t.Errorf("expected resource=secret, got %q", schema.Resource)
	}
	fieldNames := make(map[string]bool)
	for _, f := range schema.Fields {
		fieldNames[f.Name] = true
	}
	for _, want := range []string{"key", "value", "project_id"} {
		if !fieldNames[want] {
			t.Errorf("missing field %q in secret schema", want)
		}
	}
}

func TestSchemaAPIKey_OutputIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newSchemaCommand(state)
	cmd.SetArgs([]string{"api-key"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var schema schemaResource
	if err := json.Unmarshal([]byte(out), &schema); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if schema.Resource != "api_key" {
		t.Errorf("expected resource=api_key, got %q", schema.Resource)
	}
	fieldNames := make(map[string]bool)
	for _, f := range schema.Fields {
		fieldNames[f.Name] = true
	}
	for _, want := range []string{"name", "prefix", "scopes", "expires_at"} {
		if !fieldNames[want] {
			t.Errorf("missing field %q in api-key schema", want)
		}
	}
}

func TestSchemaCommand_RegisteredOnRoot(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	if !names["schema"] {
		t.Fatal("'schema' command not registered on root")
	}
}
