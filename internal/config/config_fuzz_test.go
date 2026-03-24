package config

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzParseConfigYAML(f *testing.F) {
	f.Add([]byte("server: https://api.strait.run\napi_key: sk-test-123\nproject: proj-1\n"))
	f.Add([]byte("server: https://api.strait.run\nformat: json\n"))
	f.Add([]byte("aliases:\n  ls: jobs list\n  deploy: deploy --env production\n"))
	f.Add([]byte("contexts:\n  prod:\n    server: https://api.strait.run\n    project: proj-prod\n  dev:\n    server: http://localhost:8080\n    project: proj-dev\nactive_context: prod\n"))
	f.Add([]byte("secrets:\n  proj-1:\n    - DB_URL\n    - API_KEY\n"))
	f.Add([]byte(""))
	f.Add([]byte("   \n\t  "))
	f.Add([]byte("not yaml at all {{"))
	f.Add([]byte("{}\n"))
	f.Add([]byte("\x00\xff"))
	f.Add([]byte("server: \"\"\napi_key: \"\"\n"))
	f.Add([]byte("format: yaml\n"))
	f.Add([]byte("active_context: nonexistent\ncontexts:\n  prod:\n    server: https://api.example.com\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(configPath, data, 0o600); err != nil {
			t.Fatal(err)
		}

		result, err := Load(configPath)
		if err != nil {
			return
		}

		// If loading succeeded, verify normalize guarantees.
		if result.Data == nil {
			t.Fatal("Load returned nil Data")
		}
		if result.Data.Aliases == nil {
			t.Fatal("Load returned nil Aliases map (normalize should prevent this)")
		}
		if result.Data.Secrets == nil {
			t.Fatal("Load returned nil Secrets map (normalize should prevent this)")
		}
		if result.Data.Contexts == nil {
			t.Fatal("Load returned nil Contexts map (normalize should prevent this)")
		}
	})
}
