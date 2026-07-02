package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLaunchContract_NoRetiredSurfacesInProductionCode(t *testing.T) {
	t.Parallel()

	forbidden := []string{
		`path.Join("/v1/jobs", jobID, "deployments")`,
		`path.Join("/v1/runs", runID, "tool-calls")`,
		`path.Join("/v1/runs", runID, "usage")`,
		`"/v1/billing/usage"`,
		`newDeploySourceCommand`,
		`newCodeDeploymentsCommand`,
		`Use:   "install <source>"`,
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))

	for _, root := range []string{filepath.Join(repoRoot, "cmd/strait"), filepath.Join(repoRoot, "internal/client")} {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			text := string(data)
			for _, needle := range forbidden {
				if strings.Contains(text, needle) {
					t.Errorf("%s contains retired launch surface %q", path, needle)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
}
