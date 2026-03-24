package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func FuzzParseSSEStream(f *testing.F) {
	f.Add("data: {\"type\":\"event\",\"message\":\"hello\",\"timestamp\":\"2026-03-19T10:00:00Z\"}\n\n")
	f.Add("event: status\ndata: {\"type\":\"status\",\"message\":\"ok\",\"timestamp\":\"2026-03-19T10:00:00Z\"}\n\n")
	f.Add(": comment line\ndata: {\"type\":\"event\",\"timestamp\":\"2026-03-19T10:00:00Z\"}\n\n")
	f.Add("data: {not-json}\n\n")
	f.Add("\n\n")
	f.Add("")
	f.Add("data: {\"type\":\"a\",\"timestamp\":\"2026-03-19T10:00:00Z\"}\n\ndata: {\"type\":\"b\",\"timestamp\":\"2026-03-19T10:00:00Z\"}\n\n")
	f.Add("data: line1\ndata: line2\n\n")
	f.Add("event: error\ndata: {\"error\":\"something failed\",\"timestamp\":\"2026-03-19T10:00:00Z\"}\n\n")
	f.Add("event: error\ndata: {\"type\":\"err\",\"timestamp\":\"2026-03-19T10:00:00Z\"}\n\n")
	f.Add(strings.Repeat("data: x\n", 1000) + "\n")
	f.Add("data:no-space-after-colon\n\n")
	f.Add("data: \n\n")
	f.Add("unknown: field\n\n")
	f.Add("event:\ndata: {\"type\":\"e\",\"timestamp\":\"2026-03-19T10:00:00Z\"}\n\n")

	f.Fuzz(func(t *testing.T, body string) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
		}))
		defer srv.Close()

		c := &Client{
			baseURL:    srv.URL,
			streamHTTP: srv.Client(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Must not panic or hang.
		_ = c.StreamRunEvents(ctx, "fuzz-run", func(_ RunStreamMessage) error {
			return nil
		})
	})
}

func FuzzTrimSSEField(f *testing.F) {
	f.Add("data: hello")
	f.Add("data:hello")
	f.Add("data: ")
	f.Add("data:")
	f.Add("")
	f.Add("data:  two-spaces")
	f.Add("data: {\"key\":\"value\"}")
	f.Add("data:" + strings.Repeat("x", 10000))

	f.Fuzz(func(t *testing.T, line string) {
		// Must not panic.
		result := trimSSEField(line)

		// If the line started with "data:", the result should have that prefix removed.
		if withoutPrefix, ok := strings.CutPrefix(line, "data:"); ok {
			if after, found := strings.CutPrefix(withoutPrefix, " "); found {
				if result != after {
					t.Fatalf("trimSSEField(%q) = %q, expected %q", line, result, after)
				}
			} else if result != withoutPrefix {
				t.Fatalf("trimSSEField(%q) = %q, expected %q", line, result, withoutPrefix)
			}
		}
	})
}
