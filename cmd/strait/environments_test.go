package main

import (
	"strings"
	"testing"
)

func TestParseKeyValuePairs(t *testing.T) {
	t.Parallel()

	t.Run("simple pair", func(t *testing.T) {
		t.Parallel()
		got, err := parseKeyValuePairs([]string{"FOO=bar"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "bar" {
			t.Fatalf("FOO: got %q, want %q", got["FOO"], "bar")
		}
	})

	t.Run("trims surrounding whitespace from key", func(t *testing.T) {
		t.Parallel()
		got, err := parseKeyValuePairs([]string{"  FOO  =bar"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := got["FOO"]; !ok {
			t.Fatalf("expected trimmed key FOO, got map: %#v", got)
		}
		if _, ok := got["  FOO  "]; ok {
			t.Fatal("untrimmed key was preserved; should have been trimmed")
		}
	})

	t.Run("preserves whitespace in value", func(t *testing.T) {
		t.Parallel()
		got, err := parseKeyValuePairs([]string{"GREETING=  hello world  "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["GREETING"] != "  hello world  " {
			t.Fatalf("value should not be trimmed: got %q", got["GREETING"])
		}
	})

	t.Run("empty value allowed", func(t *testing.T) {
		t.Parallel()
		got, err := parseKeyValuePairs([]string{"FOO="})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v, ok := got["FOO"]; !ok || v != "" {
			t.Fatalf("FOO=: got %q ok=%v, want empty value present", v, ok)
		}
	})

	t.Run("value contains equals", func(t *testing.T) {
		t.Parallel()
		got, err := parseKeyValuePairs([]string{"DATABASE_URL=postgres://u:p@h/db?sslmode=require"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["DATABASE_URL"] != "postgres://u:p@h/db?sslmode=require" {
			t.Fatalf("value with embedded =: got %q", got["DATABASE_URL"])
		}
	})

	t.Run("rejects missing equals", func(t *testing.T) {
		t.Parallel()
		_, err := parseKeyValuePairs([]string{"NOT_A_PAIR"})
		if err == nil {
			t.Fatal("expected error for missing =")
		}
		if !strings.Contains(err.Error(), "missing") {
			t.Fatalf("error should mention missing =: %v", err)
		}
	})

	t.Run("rejects empty key", func(t *testing.T) {
		t.Parallel()
		_, err := parseKeyValuePairs([]string{"=value"})
		if err == nil {
			t.Fatal("expected error for empty key")
		}
		if !strings.Contains(err.Error(), "empty key") {
			t.Fatalf("error should mention empty key: %v", err)
		}
	})

	t.Run("rejects whitespace-only key", func(t *testing.T) {
		t.Parallel()
		_, err := parseKeyValuePairs([]string{"   =value"})
		if err == nil {
			t.Fatal("expected error for whitespace-only key")
		}
	})

	t.Run("rejects duplicate key", func(t *testing.T) {
		t.Parallel()
		_, err := parseKeyValuePairs([]string{"FOO=a", "FOO=b"})
		if err == nil {
			t.Fatal("expected error for duplicate key")
		}
		if !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("error should mention duplicate: %v", err)
		}
	})

	t.Run("rejects duplicate after trim", func(t *testing.T) {
		t.Parallel()
		_, err := parseKeyValuePairs([]string{"FOO=a", "  FOO  =b"})
		if err == nil {
			t.Fatal("expected error for duplicate key after trim")
		}
	})

	t.Run("rejects newline in key", func(t *testing.T) {
		t.Parallel()
		_, err := parseKeyValuePairs([]string{"FOO\nBAR=v"})
		if err == nil {
			t.Fatal("expected error for newline in key")
		}
		if !strings.Contains(err.Error(), "control") {
			t.Fatalf("error should mention control char: %v", err)
		}
	})

	t.Run("rejects tab in key", func(t *testing.T) {
		t.Parallel()
		_, err := parseKeyValuePairs([]string{"FOO\tBAR=v"})
		if err == nil {
			t.Fatal("expected error for tab in key")
		}
	})

	t.Run("rejects null byte in key", func(t *testing.T) {
		t.Parallel()
		_, err := parseKeyValuePairs([]string{"FOO\x00BAR=v"})
		if err == nil {
			t.Fatal("expected error for null byte in key")
		}
	})

	t.Run("rejects DEL char in key", func(t *testing.T) {
		t.Parallel()
		_, err := parseKeyValuePairs([]string{"FOO\x7fBAR=v"})
		if err == nil {
			t.Fatal("expected error for DEL byte in key")
		}
	})

	t.Run("allows newline in value", func(t *testing.T) {
		t.Parallel()
		got, err := parseKeyValuePairs([]string{"PEM=line1\nline2"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["PEM"] != "line1\nline2" {
			t.Fatalf("multi-line value: got %q", got["PEM"])
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		got, err := parseKeyValuePairs(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("empty input: got %d entries", len(got))
		}
	})

	t.Run("multiple distinct pairs", func(t *testing.T) {
		t.Parallel()
		got, err := parseKeyValuePairs([]string{"A=1", "B=2", "C=3"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 || got["A"] != "1" || got["B"] != "2" || got["C"] != "3" {
			t.Fatalf("got %#v", got)
		}
	})
}
