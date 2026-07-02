package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizeLegacyArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "subcommand passthrough", in: []string{"version"}, want: []string{"version"}},
		{name: "flags passthrough", in: []string{"--verbose"}, want: []string{"--verbose"}},
		{name: "unknown arg passthrough", in: []string{"unknown-cmd"}, want: []string{"unknown-cmd"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeLegacyArgs(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("len(got)=%d len(want)=%d", len(got), len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("arg[%d]=%q want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestVersionCommand_UsesGlobalJSONFormat(t *testing.T) {
	t.Parallel()

	state := &appState{
		opts: &rootOptions{
			outputFormat: "json",
			timeout:      10 * time.Second,
			ciMode:       true,
			noColor:      true,
		},
		stdout: &bytes.Buffer{},
	}
	cmd := newVersionCommand(state)

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("version command: %v", err)
		}
	})

	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("version --format json output is not valid JSON: %v\noutput: %s", err, out)
	}
	if got["version"] == "" || got["commit"] == "" || got["go"] == "" || got["os_arch"] == "" {
		t.Fatalf("missing version fields: %#v", got)
	}
}
