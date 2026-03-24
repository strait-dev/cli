package output

import (
	"bytes"
	"strings"
	"testing"
)

func FuzzRenderTemplate(f *testing.F) {
	f.Add("{{.Name}}")
	f.Add("{{.Missing}}")
	f.Add("")
	f.Add("no template syntax")
	f.Add("{{range .Items}}{{.}}{{end}}")
	f.Add("{{if .Name}}yes{{else}}no{{end}}")
	f.Add("{{len .Name}}")
	f.Add("{{printf \"%s\" .Name}}")
	f.Add("{{.Name | printf \"%q\"}}")
	f.Add("{{ }}")
	f.Add("{{")
	f.Add("}}")
	f.Add("{{.}}")
	f.Add(strings.Repeat("{{.Name}}", 100))

	data := map[string]any{
		"Name":  "test-project",
		"ID":    "proj-123",
		"Items": []string{"a", "b", "c"},
		"Count": 42,
	}

	f.Fuzz(func(t *testing.T, tpl string) {
		var buf bytes.Buffer
		// Must not panic.
		_ = renderTemplate(&buf, data, tpl)
	})
}

func FuzzRenderJSONPath(f *testing.F) {
	f.Add("$.name")
	f.Add("$")
	f.Add("$.items[0]")
	f.Add("$.missing")
	f.Add("$..name")
	f.Add("")
	f.Add("$[0]")
	f.Add("$.items[*]")
	f.Add("$.nested.deep.value")
	f.Add("invalid")
	f.Add("$.")
	f.Add("$[")
	f.Add(strings.Repeat("$.a", 100))

	data := map[string]any{
		"name": "test-project",
		"id":   "proj-123",
		"items": []any{
			map[string]any{"name": "item1", "value": 1},
			map[string]any{"name": "item2", "value": 2},
		},
		"nested": map[string]any{
			"deep": map[string]any{
				"value": "found",
			},
		},
		"count": 42,
	}

	f.Fuzz(func(t *testing.T, expr string) {
		var buf bytes.Buffer
		// Must not panic.
		_ = renderJSONPath(&buf, data, expr)
	})
}

func FuzzRenderCSV(f *testing.F) {
	f.Add("name", "value")
	f.Add("with,comma", "with\"quote")
	f.Add("with\nnewline", "with\ttab")
	f.Add("", "")
	f.Add("normal-key", "normal-value")
	f.Add("key with spaces", "value with spaces")
	f.Add("\x00", "\xff")
	f.Add("a\"b\"c", "d,e,f")

	f.Fuzz(func(t *testing.T, key, value string) {
		data := []map[string]string{
			{key: value},
		}
		var buf bytes.Buffer
		// Must not panic.
		_ = renderCSV(&buf, data)
	})
}

func FuzzRender(f *testing.F) {
	f.Add("json")
	f.Add("yaml")
	f.Add("table")
	f.Add("csv")
	f.Add("wide")
	f.Add("go-template")
	f.Add("jsonpath")
	f.Add("")
	f.Add("unknown")
	f.Add("JSON")
	f.Add("  json  ")
	f.Add(strings.Repeat("x", 1000))

	data := map[string]string{
		"name": "test",
		"id":   "123",
	}

	f.Fuzz(func(t *testing.T, format string) {
		var buf bytes.Buffer
		opts := Options{
			Format:   format,
			Template: "{{.name}}",
			JSONPath: "$.name",
			TTY:      false,
		}
		// Must not panic.
		_ = Render(&buf, data, opts)
	})
}
