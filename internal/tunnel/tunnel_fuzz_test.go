package tunnel

import (
	"regexp"
	"strings"
	"testing"
)

var tunnelURLCheck = regexp.MustCompile(`^https://[a-zA-Z0-9-]+\.trycloudflare\.com$`)

func FuzzParseTunnelURL(f *testing.F) {
	f.Add("2026/03/19 10:00:00 https://some-random-words.trycloudflare.com registered")
	f.Add("https://abc-def-123.trycloudflare.com")
	f.Add("no tunnel url here")
	f.Add("")
	f.Add("https://example.com")
	f.Add("http://abc.trycloudflare.com")
	f.Add("multiple https://first.trycloudflare.com and https://second.trycloudflare.com urls")
	f.Add("https://.trycloudflare.com")
	f.Add("https://trycloudflare.com")
	f.Add(strings.Repeat("x", 10000))
	f.Add("https://a.trycloudflare.com\nhttps://b.trycloudflare.com")

	f.Fuzz(func(t *testing.T, output string) {
		result, err := ParseTunnelURL(output)
		if err != nil {
			return
		}

		// If a URL was found, it must match the expected pattern.
		if !tunnelURLCheck.MatchString(result) {
			t.Fatalf("ParseTunnelURL returned non-matching URL: %q", result)
		}
	})
}

func FuzzBuildJobEndpoints(f *testing.F) {
	f.Add("https://abc.trycloudflare.com", "api", "/webhook")
	f.Add("https://abc.trycloudflare.com/", "worker", "")
	f.Add("https://abc.trycloudflare.com", "etl", "custom/path")
	f.Add("https://abc.trycloudflare.com", "job", "/")
	f.Add("http://localhost:8080", "api", "/hook")
	f.Add("", "", "")
	f.Add("https://example.com", "s", "/p")

	f.Fuzz(func(t *testing.T, tunnelURL, slug, path string) {
		jobs := []JobEndpoint{{Slug: slug, Path: path}}

		// Must not panic.
		endpoints := BuildJobEndpoints(tunnelURL, jobs)

		if len(endpoints) != 1 {
			t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
		}

		endpoint, ok := endpoints[slug]
		if !ok {
			t.Fatalf("endpoint for slug %q not found", slug)
		}

		// The endpoint must start with the base URL (trimmed of trailing slash).
		base := strings.TrimRight(tunnelURL, "/")
		if !strings.HasPrefix(endpoint, base) {
			t.Fatalf("endpoint %q does not start with base %q", endpoint, base)
		}
	})
}
