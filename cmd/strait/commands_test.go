package main

import "testing"

func TestParseWaitCondition(t *testing.T) {
	t.Parallel()

	status, err := parseWaitCondition("status=completed")
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if status != "completed" {
		t.Fatalf("status mismatch: %s", status)
	}
}
