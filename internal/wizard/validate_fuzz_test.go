package wizard

import (
	"math"
	"net"
	"net/url"
	"strings"
	"testing"
)

func FuzzValidateProjectName(f *testing.F) {
	f.Add("my-api")
	f.Add("a")
	f.Add("api123")
	f.Add("my-cool-api")
	f.Add("")
	f.Add("-bad")
	f.Add("bad-")
	f.Add("MY-API")
	f.Add("my_api")
	f.Add(strings.Repeat("a", 128))
	f.Add(strings.Repeat("a", 129))
	f.Add("hello world")
	f.Add("\t\n\r")
	f.Add("\x00")
	f.Add("a-b-c-d-e")

	f.Fuzz(func(t *testing.T, name string) {
		// Must never panic.
		_ = ValidateProjectName(name)
	})
}

func FuzzValidateSlug(f *testing.F) {
	f.Add("my-api")
	f.Add("a")
	f.Add("api123")
	f.Add("")
	f.Add("-bad")
	f.Add("bad-")
	f.Add("MY-API")
	f.Add("my_api")
	f.Add(strings.Repeat("a", 128))
	f.Add(strings.Repeat("a", 129))
	f.Add("hello world")
	f.Add("\t\n")
	f.Add("\x00")

	f.Fuzz(func(t *testing.T, slug string) {
		// Must never panic.
		_ = ValidateSlug(slug)
	})
}

func FuzzValidateEndpoint(f *testing.F) {
	f.Add("https://api.example.com/jobs/process")
	f.Add("http://localhost:8080/webhook")
	f.Add("http://127.0.0.1:3000")
	f.Add("https://10.0.0.1/internal")
	f.Add("https://192.168.1.1/internal")
	f.Add("https://169.254.169.254/latest/meta-data")
	f.Add("https://metadata.google.internal/v1")
	f.Add("ftp://example.com/file")
	f.Add("")
	f.Add("not-a-url")
	f.Add("http://[::1]:8080/path")
	f.Add("http://100.64.0.1/cgnat")
	f.Add("https://user:pass@example.com/path")
	f.Add(strings.Repeat("a", 2049))
	f.Add("http://")
	f.Add("https://")
	f.Add("://missing-scheme")
	f.Add("https://example.com:99999/path")

	f.Fuzz(func(t *testing.T, endpoint string) {
		err := ValidateEndpoint(endpoint)
		if err != nil {
			return
		}

		// ValidateEndpoint trims the input, so check invariants on the trimmed value.
		trimmed := strings.TrimSpace(endpoint)
		parsed, parseErr := url.Parse(trimmed)
		if parseErr != nil {
			t.Fatalf("ValidateEndpoint accepted unparseable URL: %q", trimmed)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			t.Fatalf("ValidateEndpoint accepted non-http(s) scheme: %q", parsed.Scheme)
		}
		if parsed.Host == "" {
			t.Fatalf("ValidateEndpoint accepted URL with empty host: %q", trimmed)
		}

		hostname := parsed.Hostname()
		if isBlockedHost(hostname) {
			t.Fatalf("ValidateEndpoint accepted blocked host: %q", hostname)
		}
		if ip := net.ParseIP(hostname); ip != nil {
			if isPrivateIP(ip) {
				t.Fatalf("ValidateEndpoint accepted private IP: %q", hostname)
			}
		}
	})
}

func FuzzValidateCron(f *testing.F) {
	f.Add("*/5 * * * *")
	f.Add("0 12 * * 1-5")
	f.Add("@hourly")
	f.Add("@daily")
	f.Add("@yearly")
	f.Add("@annually")
	f.Add("@monthly")
	f.Add("@weekly")
	f.Add("@midnight")
	f.Add("")
	f.Add("* * *")
	f.Add("* * * * * *")
	f.Add("1 2 3 4 5 6")
	f.Add("not a cron")
	f.Add("* * * * * * *")
	f.Add("\t  \n")

	f.Fuzz(func(t *testing.T, expr string) {
		err := ValidateCron(expr)
		if err != nil {
			return
		}

		// If accepted, verify it's either empty, a known alias, or has 5-6 fields.
		trimmed := strings.TrimSpace(expr)
		if trimmed == "" {
			return
		}
		aliases := map[string]bool{
			"@yearly": true, "@annually": true,
			"@monthly": true, "@weekly": true,
			"@daily": true, "@midnight": true,
			"@hourly": true,
		}
		if aliases[trimmed] {
			return
		}
		parts := strings.Fields(trimmed)
		if len(parts) != 5 && len(parts) != 6 {
			t.Fatalf("ValidateCron accepted expression with %d fields: %q", len(parts), expr)
		}
	})
}

func FuzzValidateRuntime(f *testing.F) {
	f.Add("node")
	f.Add("bun")
	f.Add("python")
	f.Add("go")
	f.Add("docker")
	f.Add("")
	f.Add("DOCKER")
	f.Add("Node")
	f.Add("rust")
	f.Add("java")
	f.Add("  go  ")

	f.Fuzz(func(t *testing.T, runtime string) {
		err := ValidateRuntime(runtime)
		if err != nil {
			return
		}

		// If accepted, the lowered/trimmed value must be one of the valid runtimes.
		normalized := strings.TrimSpace(strings.ToLower(runtime))
		valid := map[string]bool{
			"go": true, "python": true, "typescript": true,
			"ruby": true, "rust": true, "node": true, "bun": true, "docker": true,
		}
		if !valid[normalized] {
			t.Fatalf("ValidateRuntime accepted invalid runtime: %q (normalized: %q)", runtime, normalized)
		}
	})
}

func FuzzValidateTimeout(f *testing.F) {
	f.Add(0)
	f.Add(1)
	f.Add(86400)
	f.Add(86401)
	f.Add(-1)
	f.Add(math.MaxInt32)
	f.Add(math.MinInt32)
	f.Add(43200)
	f.Add(2)

	f.Fuzz(func(t *testing.T, secs int) {
		err := ValidateTimeout(secs)
		if err != nil {
			return
		}

		// If accepted, must be in [1, 86400].
		if secs < 1 || secs > 86400 {
			t.Fatalf("ValidateTimeout accepted out-of-range value: %d", secs)
		}
	})
}

func FuzzValidateMaxAttempts(f *testing.F) {
	f.Add(0)
	f.Add(1)
	f.Add(100)
	f.Add(101)
	f.Add(-1)
	f.Add(math.MaxInt32)
	f.Add(math.MinInt32)
	f.Add(50)

	f.Fuzz(func(t *testing.T, n int) {
		err := ValidateMaxAttempts(n)
		if err != nil {
			return
		}

		// If accepted, must be in [1, 100].
		if n < 1 || n > 100 {
			t.Fatalf("ValidateMaxAttempts accepted out-of-range value: %d", n)
		}
	})
}
