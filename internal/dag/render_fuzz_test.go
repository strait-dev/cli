package dag

import (
	"strings"
	"testing"
)

func FuzzRenderDAG(f *testing.F) {
	// Encoding: steps separated by ";", each step is "ref" or "ref:dep1,dep2"
	f.Add("a")
	f.Add("a;b:a")
	f.Add("a;b:a;c:a")
	f.Add("a;b:a;c:b")
	f.Add("a;b:a;c:a,b")
	f.Add("a;b:a;c:b;d:b,c")
	f.Add("a;b:b")                // self-reference
	f.Add("a;b:c")                // dangling reference
	f.Add("a;b:a;a:b")            // cycle
	f.Add("")                     // empty
	f.Add("step-1;step-2:step-1") // hyphens in names
	f.Add(strings.Repeat("s;", 50) + "s")

	f.Fuzz(func(t *testing.T, encoded string) {
		if encoded == "" {
			steps := []Step{}
			result := RenderDAG(steps, nil)
			if result == "" {
				t.Fatal("RenderDAG returned empty string for empty steps")
			}
			return
		}

		parts := strings.Split(encoded, ";")
		steps := make([]Step, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			ref, depsStr, hasDeps := strings.Cut(part, ":")
			if !hasDeps {
				steps = append(steps, Step{StepRef: ref})
			} else {
				var deps []string
				if depsStr != "" {
					deps = strings.Split(depsStr, ",")
				}
				steps = append(steps, Step{StepRef: ref, DependsOn: deps})
			}
		}

		if len(steps) == 0 {
			return
		}

		// Must not panic.
		result := RenderDAG(steps, nil)
		if result == "" {
			t.Fatal("RenderDAG returned empty string")
		}

		// Also test with a status map.
		statusMap := map[string]string{}
		for _, s := range steps {
			statusMap[s.StepRef] = "completed"
		}
		resultWithStatus := RenderDAG(steps, statusMap)
		if resultWithStatus == "" {
			t.Fatal("RenderDAG with status map returned empty string")
		}
	})
}
