package ci

import (
	"strings"
	"testing"
)

func FuzzContainsUnsafeChars(f *testing.F) {
	f.Add("safe-string")
	f.Add("my-proj_123.test")
	f.Add("proj-${BAD}")
	f.Add("`whoami`")
	f.Add("{{.Evil}}")
	f.Add("$(cmd)")
	f.Add("a;b")
	f.Add("a|b")
	f.Add("a&b")
	f.Add("a\nb")
	f.Add("a\rb")
	f.Add("")
	f.Add("   ")
	f.Add("normal")
	f.Add("${{secrets.KEY}}")
	f.Add(strings.Repeat("safe", 1000))

	unsafePatterns := []string{"${", "`", "{{", "}}", "$(", ";", "|", "&", "\n", "\r"}

	f.Fuzz(func(t *testing.T, s string) {
		result := containsUnsafeChars(s)

		// If the function says it's safe, verify none of the unsafe patterns are present.
		if !result {
			for _, pat := range unsafePatterns {
				if strings.Contains(s, pat) {
					t.Fatalf("containsUnsafeChars(%q) returned false, but string contains %q", s, pat)
				}
			}
		}

		// If the function says it's unsafe, at least one pattern must be present.
		if result {
			found := false
			for _, pat := range unsafePatterns {
				if strings.Contains(s, pat) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("containsUnsafeChars(%q) returned true, but no unsafe pattern found", s)
			}
		}
	})
}

func FuzzGenerate(f *testing.F) {
	f.Add("my-project", "production")
	f.Add("my-project", "staging")
	f.Add("proj-${BAD}", "prod")
	f.Add("safe", "env-`whoami`")
	f.Add("safe", "$(bad)")
	f.Add("safe", "a;b")
	f.Add("", "")
	f.Add("proj", "")
	f.Add("", "env")
	f.Add("a\nb", "c\rd")
	f.Add("proj-123", "dev")
	f.Add("{{.Inject}}", "env")
	f.Add("safe", "{{.Inject}}")

	providers := []string{"github", "gitlab", "generic"}

	f.Fuzz(func(t *testing.T, projectID, environment string) {
		for _, provider := range providers {
			cfg := GenerateConfig{
				ProjectID:   projectID,
				Environment: environment,
			}
			output, err := Generate(provider, cfg)
			if err != nil {
				continue
			}

			// If generation succeeded, the output should contain the provided values.
			if !strings.Contains(output, projectID) {
				t.Fatalf("Generate(%q, %+v) output does not contain projectID %q", provider, cfg, projectID)
			}
			if !strings.Contains(output, environment) {
				t.Fatalf("Generate(%q, %+v) output does not contain environment %q", provider, cfg, environment)
			}
		}
	})
}
