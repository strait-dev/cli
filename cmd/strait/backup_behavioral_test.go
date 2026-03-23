package main

import (
	"os"
	"strings"
	"testing"
)

func TestBackupCreate_NoDatabaseURL(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{ciMode: true}}
	cmd := newBackupCreateCommand(state)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	// May fail with "pg_dump not found" (no postgres tools) or "database URL required".
	msg := err.Error()
	if !strings.Contains(msg, "database URL required") && !strings.Contains(msg, "pg_dump not found") {
		t.Fatalf("expected database URL or pg_dump error, got: %v", err)
	}
}

func TestBackupRestore_MissingInput(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{ciMode: true}}
	cmd := newBackupRestoreCommand(state)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--input is required") {
		t.Fatalf("expected --input required error, got: %v", err)
	}
}

func TestBackupRestore_FileNotFound(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{ciMode: true}}
	cmd := newBackupRestoreCommand(state)
	cmd.SetArgs([]string{"--input", "/tmp/nonexistent-backup-file.sql", "--database-url", "postgres://test:test@localhost:5432/test"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected file not found error, got: %v", err)
	}
}

func TestResolveDatabaseURL_FromFlag(t *testing.T) {
	t.Parallel()

	got := resolveDatabaseURL("postgres://user:pass@host:5432/db")
	if got != "postgres://user:pass@host:5432/db" {
		t.Fatalf("expected flag value, got: %s", got)
	}
}

func TestResolveDatabaseURL_InvalidScheme(t *testing.T) {
	t.Parallel()

	got := resolveDatabaseURL("mysql://host")
	if got != "" {
		t.Fatalf("expected empty for invalid scheme, got: %s", got)
	}
}

func TestResolveDatabaseURL_Empty(t *testing.T) {
	// Cannot use t.Parallel with t.Setenv, but this test is fast.
	t.Setenv("DATABASE_URL", "")
	got := resolveDatabaseURL("")
	if got != "" {
		t.Fatalf("expected empty, got: %s", got)
	}
}

func TestIsPlainSQL_DetectsCustomFormat(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/test.dump"
	if err := os.WriteFile(path, []byte("PGDMP1234567890"), 0o600); err != nil {
		t.Fatal(err)
	}

	if isPlainSQL(path) {
		t.Fatal("expected custom format (not plain SQL) for PGDMP file")
	}
}

func TestIsPlainSQL_DetectsPlain(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/test.sql"
	if err := os.WriteFile(path, []byte("-- PostgreSQL dump\nSELECT 1;"), 0o600); err != nil {
		t.Fatal(err)
	}

	if !isPlainSQL(path) {
		t.Fatal("expected plain SQL detection for text file")
	}
}

func TestIsPlainSQL_NonexistentFile(t *testing.T) {
	t.Parallel()

	// Nonexistent file defaults to plain SQL (psql)
	if !isPlainSQL("/nonexistent/file.sql") {
		t.Fatal("expected default to plain SQL for missing file")
	}
}
