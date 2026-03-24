package extension

import (
	"strings"
	"testing"
)

func FuzzParseManifest(f *testing.F) {
	f.Add([]byte(`{"name":"my-plugin","version":"1.0.0","commands":["greet"]}`))
	f.Add([]byte(`{"name":"my-plugin","version":"1.0.0","commands":["greet"],"hooks":["pre-deploy"]}`))
	f.Add([]byte(`{"name":"my-plugin","version":"1.0.0","commands":["greet"],"hooks":["pre-deploy","post-deploy"]}`))
	f.Add([]byte(`{"name":"","version":"1.0.0","commands":["greet"]}`))
	f.Add([]byte(`{"name":"p","version":"1.0.0","commands":[]}`))
	f.Add([]byte(`{"name":"p","version":"1.0.0","commands":["c"],"hooks":["unknown-hook"]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"name":"p"}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(``))
	f.Add([]byte(`{"name":"p","version":"1.0.0","commands":["a","b","c"]}`))
	f.Add([]byte("\x00\xff\xfe"))
	f.Add([]byte(`{"name":"` + strings.Repeat("x", 10000) + `","version":"1.0.0","commands":["c"]}`))
	f.Add([]byte(`{"name":"p","version":"1.0.0","commands":["c"],"hooks":["pre-deploy","pre-deploy"]}`))
	f.Add([]byte(`{"name":"p","version":"1.0.0","commands":["c"],"description":"desc","extra":"field"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		m, err := ParseManifest(data)
		if err != nil {
			return
		}

		// If parsing succeeded, validate invariants.
		if strings.TrimSpace(m.Name) == "" {
			t.Fatal("ParseManifest returned manifest with empty name")
		}
		if len(m.Commands) == 0 {
			t.Fatal("ParseManifest returned manifest with no commands")
		}

		known := make(map[string]bool, len(KnownHooks))
		for _, h := range KnownHooks {
			known[h] = true
		}
		for _, h := range m.Hooks {
			if !known[h] {
				t.Fatalf("ParseManifest returned manifest with unknown hook: %q", h)
			}
		}
	})
}

func FuzzValidateManifest(f *testing.F) {
	f.Add("my-plugin", "1.0.0", "greet", "")
	f.Add("my-plugin", "1.0.0", "greet,run", "pre-deploy")
	f.Add("", "1.0.0", "greet", "")
	f.Add("p", "1.0.0", "", "")
	f.Add("p", "1.0.0", "c", "unknown-hook")
	f.Add("p", "", "c", "pre-deploy,post-deploy")
	f.Add("  ", "1.0.0", "c", "")
	f.Add("p", "1.0.0", "c", "pre-deploy,pre-deploy")

	f.Fuzz(func(t *testing.T, name, version, commandsStr, hooksStr string) {
		var commands []string
		if commandsStr != "" {
			commands = strings.Split(commandsStr, ",")
		}
		var hooks []string
		if hooksStr != "" {
			hooks = strings.Split(hooksStr, ",")
		}

		m := &PluginManifest{
			Name:     name,
			Version:  version,
			Commands: commands,
			Hooks:    hooks,
		}

		// Must never panic.
		_ = ValidateManifest(m)
	})
}
