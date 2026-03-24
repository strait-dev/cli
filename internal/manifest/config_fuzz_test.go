package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func FuzzParseProjectConfigJSON(f *testing.F) {
	f.Add([]byte(`{"project":{"id":"proj-123","name":"My Project"}}`))
	f.Add([]byte(`{"project":{"id":"proj-123"},"jobs":[{"slug":"etl","name":"ETL Job"}]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"project":{}}`))
	f.Add([]byte(`{"project":{"id":""}}`))
	f.Add([]byte(`not json at all`))
	f.Add([]byte(``))
	f.Add([]byte(`   `))
	f.Add([]byte(`{"project":{"id":"x"},"jobs":[{"slug":"","name":"bad"}]}`))
	f.Add([]byte(`{"project":{"id":"x"},"workflows":[{"slug":"w","name":"W"}]}`))
	f.Add([]byte(`{"project":{"id":"x"},"jobs":null}`))
	f.Add([]byte(`{"project":{"id":"x","name":"p"},"runtime":"node"}`))
	f.Add([]byte("\x00\xff\xfe"))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "strait.json")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadProjectConfig(path)
		if err != nil {
			return
		}
		assertConfigInvariants(t, cfg)
	})
}

func FuzzParseProjectConfigYAML(f *testing.F) {
	f.Add([]byte("project:\n  id: proj-123\n  name: My Project\n"))
	f.Add([]byte("project:\n  id: proj-123\njobs:\n  - slug: etl\n    name: ETL Job\n"))
	f.Add([]byte("project:\n  id: x\n"))
	f.Add([]byte(""))
	f.Add([]byte("   \n  \t  "))
	f.Add([]byte("invalid: [yaml: {{"))
	f.Add([]byte("project:\n  id: \"\"\n"))
	f.Add([]byte("---\nproject:\n  id: multi-doc\n"))
	f.Add([]byte("\x00\xff"))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "strait.config.yaml")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadProjectConfig(path)
		if err != nil {
			return
		}
		assertConfigInvariants(t, cfg)
	})
}

func FuzzParseProjectConfigUnknownExt(f *testing.F) {
	f.Add([]byte(`{"project":{"id":"proj-json"}}`))
	f.Add([]byte("project:\n  id: proj-yaml\n"))
	f.Add([]byte(`not json and not yaml either {{`))
	f.Add([]byte(`{"project":{"id":"x"},"jobs":[{"slug":"s","name":"n"}]}`))
	f.Add([]byte(""))
	f.Add([]byte("   "))
	f.Add([]byte("\x00"))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.conf")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadProjectConfig(path)
		if err != nil {
			return
		}
		assertConfigInvariants(t, cfg)
	})
}

func assertConfigInvariants(t *testing.T, cfg *ProjectConfig) {
	t.Helper()
	if cfg.Project.ID == "" {
		t.Fatal("LoadProjectConfig returned config with empty project.id")
	}
	for i, j := range cfg.Jobs {
		if strings.TrimSpace(j.Slug) == "" {
			t.Fatalf("LoadProjectConfig returned config with empty jobs[%d].slug", i)
		}
		if strings.TrimSpace(j.Name) == "" {
			t.Fatalf("LoadProjectConfig returned config with empty jobs[%d].name", i)
		}
	}
	for i, w := range cfg.Workflows {
		if strings.TrimSpace(w.Slug) == "" {
			t.Fatalf("LoadProjectConfig returned config with empty workflows[%d].slug", i)
		}
		if strings.TrimSpace(w.Name) == "" {
			t.Fatalf("LoadProjectConfig returned config with empty workflows[%d].name", i)
		}
	}
}
