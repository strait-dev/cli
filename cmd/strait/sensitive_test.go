package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMaskString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		in     string
		reveal bool
		want   string
	}{
		{name: "non-empty masked", in: "supersecret", reveal: false, want: sensitiveMask},
		{name: "non-empty revealed", in: "supersecret", reveal: true, want: "supersecret"},
		{name: "empty masked", in: "", reveal: false, want: ""},
		{name: "empty revealed", in: "", reveal: true, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := maskString(tc.in, tc.reveal)
			if got != tc.want {
				t.Fatalf("maskString(%q, %v): got %q, want %q", tc.in, tc.reveal, got, tc.want)
			}
		})
	}
}

func TestMaskRawJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		in        json.RawMessage
		reveal    bool
		wantMask  bool
		wantOrig  bool
		wantValid bool
	}{
		{name: "non-empty masked", in: json.RawMessage(`{"api_key":"xyz"}`), reveal: false, wantMask: true, wantValid: true},
		{name: "non-empty revealed", in: json.RawMessage(`{"api_key":"xyz"}`), reveal: true, wantOrig: true, wantValid: true},
		{name: "empty masked", in: nil, reveal: false, wantOrig: true},
		{name: "empty revealed", in: nil, reveal: true, wantOrig: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := maskRawJSON(tc.in, tc.reveal)
			if tc.wantMask {
				if string(got) != `"`+sensitiveMask+`"` {
					t.Fatalf("expected mask placeholder, got %q", string(got))
				}
				if !json.Valid(got) {
					t.Fatalf("mask is not valid JSON: %q", string(got))
				}
			}
			if tc.wantOrig && string(got) != string(tc.in) {
				t.Fatalf("expected original, got %q want %q", string(got), string(tc.in))
			}
		})
	}
}

func TestMaskMapValues(t *testing.T) {
	t.Parallel()

	t.Run("non-empty masked", func(t *testing.T) {
		t.Parallel()
		in := map[string]string{"DB_PASSWORD": "hunter2", "API_KEY": "abc123", "PUBLIC": ""}
		got := maskMapValues(in, false)
		if got["DB_PASSWORD"] != sensitiveMask {
			t.Fatalf("DB_PASSWORD: got %q, want %q", got["DB_PASSWORD"], sensitiveMask)
		}
		if got["API_KEY"] != sensitiveMask {
			t.Fatalf("API_KEY: got %q, want %q", got["API_KEY"], sensitiveMask)
		}
		// Empty values are NOT masked — masking "" would imply a secret was present.
		if got["PUBLIC"] != "" {
			t.Fatalf("PUBLIC: got %q, want empty", got["PUBLIC"])
		}
		// Original map MUST NOT be mutated.
		if in["DB_PASSWORD"] != "hunter2" {
			t.Fatalf("input mutated: got %q", in["DB_PASSWORD"])
		}
	})

	t.Run("revealed returns original", func(t *testing.T) {
		t.Parallel()
		in := map[string]string{"DB_PASSWORD": "hunter2"}
		got := maskMapValues(in, true)
		if got["DB_PASSWORD"] != "hunter2" {
			t.Fatalf("revealed: got %q, want %q", got["DB_PASSWORD"], "hunter2")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		in := map[string]string{}
		got := maskMapValues(in, false)
		if len(got) != 0 {
			t.Fatalf("empty input: got %d entries, want 0", len(got))
		}
	})
}

// TestMaskedJSONOutputDoesNotLeakSecrets sanity-checks that json.Marshal of a
// masked struct produces output that does not contain the original secret.
func TestMaskedJSONOutputDoesNotLeakSecrets(t *testing.T) {
	t.Parallel()

	type config struct {
		Name   string            `json:"name"`
		Vars   map[string]string `json:"vars"`
		Config json.RawMessage   `json:"config"`
		Secret string            `json:"secret"`
	}

	c := config{
		Name:   "prod",
		Vars:   map[string]string{"PG_PASSWORD": "very-secret-pg-password"},
		Config: json.RawMessage(`{"slack_webhook":"https://hooks.slack.com/secret-token"}`),
		Secret: "whsec_live_secret_xyz",
	}

	masked := config{
		Name:   c.Name,
		Vars:   maskMapValues(c.Vars, false),
		Config: maskRawJSON(c.Config, false),
		Secret: maskString(c.Secret, false),
	}

	out, err := json.Marshal(masked)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	leaks := []string{
		"very-secret-pg-password",
		"hooks.slack.com/secret-token",
		"whsec_live_secret_xyz",
	}
	for _, leak := range leaks {
		if strings.Contains(string(out), leak) {
			t.Fatalf("masked output leaked %q: %s", leak, string(out))
		}
	}
	if !strings.Contains(string(out), sensitiveMask) {
		t.Fatalf("masked output missing mask placeholder: %s", string(out))
	}
}
