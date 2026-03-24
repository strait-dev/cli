package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzParseDeployConfig(f *testing.F) {
	f.Add([]byte("version: 1\nproject: proj-1\njobs:\n  - slug: api\n    dockerfile: Dockerfile\n"))
	f.Add([]byte("version: 1\nproject: proj-1\njobs:\n  - slug: api\n    dockerfile: Dockerfile\n    preset: small\n    region: us-east-1\n"))
	f.Add([]byte("version: 1\nproject: proj-1\njobs:\n  - slug: api\n    dockerfile: Dockerfile\n  - slug: worker\n    dockerfile: Dockerfile.worker\n"))
	f.Add([]byte("version: 2\nproject: proj-1\njobs:\n  - slug: api\n    dockerfile: Dockerfile\n"))
	f.Add([]byte("version: 1\nproject: proj-1\njobs: []\n"))
	f.Add([]byte("version: 1\nproject: proj-1\njobs:\n  - slug: api\n    dockerfile: Dockerfile\n  - slug: api\n    dockerfile: Dockerfile2\n"))
	f.Add([]byte("version: 1\n"))
	f.Add([]byte(""))
	f.Add([]byte("   \n\t  "))
	f.Add([]byte("not yaml at all {{"))
	f.Add([]byte("{}\n"))
	f.Add([]byte("\x00\xff"))
	f.Add([]byte("version: 1\nproject: proj-1\njobs:\n  - slug: \"\"\n    dockerfile: Dockerfile\n"))
	f.Add([]byte("version: 1\nproject: proj-1\njobs:\n  - slug: api\n    dockerfile: nonexistent.Dockerfile\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()

		// Create a Dockerfile so that validation can find it.
		dockerfilePath := filepath.Join(dir, "Dockerfile")
		if err := os.WriteFile(dockerfilePath, []byte("FROM scratch\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		configPath := filepath.Join(dir, "strait.config.yaml")
		if err := os.WriteFile(configPath, data, 0o600); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadDeployConfig(configPath)
		if err != nil {
			return
		}

		// If accepted, verify invariants.
		if cfg.Version != 1 {
			t.Fatalf("LoadDeployConfig accepted config with version %d", cfg.Version)
		}
		if len(cfg.Jobs) == 0 {
			t.Fatal("LoadDeployConfig accepted config with no jobs")
		}
		seen := make(map[string]bool)
		for _, job := range cfg.Jobs {
			if job.Slug == "" {
				t.Fatal("LoadDeployConfig accepted job with empty slug")
			}
			if seen[job.Slug] {
				t.Fatalf("LoadDeployConfig accepted duplicate slug: %q", job.Slug)
			}
			seen[job.Slug] = true
		}
	})
}
