package bundle

import (
	"testing"
)

func FuzzUnmarshalYAML(f *testing.F) {
	f.Add([]byte("version: \"1\"\nexported_at: 2026-03-19T10:00:00Z\nsource_project_id: proj-1\nresources:\n  jobs: []\n"))
	f.Add([]byte("version: \"1\"\nexported_at: 2026-03-19T10:00:00Z\nsource_project_id: proj-1\n"))
	f.Add([]byte("version: \"2\"\n"))
	f.Add([]byte("version: \"\"\n"))
	f.Add([]byte(""))
	f.Add([]byte("not yaml {{"))
	f.Add([]byte("{}"))
	f.Add([]byte("\x00\xff"))
	f.Add([]byte("version: \"1\"\nsource_project_id: proj-1\nresources:\n  jobs:\n    - slug: api\n      name: API\n      endpoint_url: https://example.com\n      max_attempts: 3\n      timeout_secs: 60\n      enabled: true\n"))
	f.Add([]byte("version: \"1\"\nsource_project_id: proj-1\nresources:\n  workflows:\n    - slug: pipe\n      name: Pipeline\n      steps:\n        - step_ref: s1\n          job_slug: api\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		b, err := UnmarshalYAML(data)
		if err != nil {
			return
		}

		// If accepted, version must be the current version.
		if b.Version != Version {
			t.Fatalf("UnmarshalYAML accepted bundle with version %q, expected %q", b.Version, Version)
		}
	})
}

func FuzzMarshalUnmarshalRoundTrip(f *testing.F) {
	f.Add("proj-1", "api", "API Job", "https://example.com/webhook")
	f.Add("proj-2", "worker", "Worker", "https://example.com/worker")
	f.Add("", "", "", "")
	f.Add("proj-with-special-chars!", "slug with spaces", "Name\nwith\nnewlines", "not-a-url")
	f.Add("p", "s", "n", "u")

	f.Fuzz(func(t *testing.T, projectID, jobSlug, jobName, endpointURL string) {
		original := &Bundle{
			Version:         Version,
			SourceProjectID: projectID,
			Resources: Resources{
				Jobs: []JobSpec{
					{
						Slug:        jobSlug,
						Name:        jobName,
						EndpointURL: endpointURL,
						MaxAttempts: 3,
						TimeoutSecs: 60,
						Enabled:     true,
					},
				},
			},
		}

		data, err := MarshalYAML(original)
		if err != nil {
			return
		}

		roundTripped, err := UnmarshalYAML(data)
		if err != nil {
			t.Fatalf("round-trip failed: MarshalYAML succeeded but UnmarshalYAML failed: %v", err)
		}

		if roundTripped.Version != original.Version {
			t.Fatalf("round-trip version mismatch: got %q, want %q", roundTripped.Version, original.Version)
		}
		if roundTripped.SourceProjectID != original.SourceProjectID {
			t.Fatalf("round-trip source_project_id mismatch: got %q, want %q", roundTripped.SourceProjectID, original.SourceProjectID)
		}
		if len(roundTripped.Resources.Jobs) != len(original.Resources.Jobs) {
			t.Fatalf("round-trip jobs count mismatch: got %d, want %d", len(roundTripped.Resources.Jobs), len(original.Resources.Jobs))
		}
		if len(roundTripped.Resources.Jobs) > 0 {
			got := roundTripped.Resources.Jobs[0]
			want := original.Resources.Jobs[0]
			if got.Slug != want.Slug {
				t.Fatalf("round-trip job slug mismatch: got %q, want %q", got.Slug, want.Slug)
			}
			if got.Name != want.Name {
				t.Fatalf("round-trip job name mismatch: got %q, want %q", got.Name, want.Name)
			}
			if got.EndpointURL != want.EndpointURL {
				t.Fatalf("round-trip job endpoint_url mismatch: got %q, want %q", got.EndpointURL, want.EndpointURL)
			}
		}
	})
}
