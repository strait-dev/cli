package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunEHandlersDoNotBypassPrintData walks every cobra command literal in
// cmd/strait/*.go and asserts that the RunE function body does not write
// primary output by reaching past printData / state.out() to the global
// os.Stdout (or to fmt.Print/Println/Printf, which write to os.Stdout
// implicitly).
//
// The motivating regression: `workflows visualize` originally used
// fmt.Print(rendered) instead of routing through state.out(), so --format
// json/yaml were silently ignored. This test catches the whole class.
//
// Allowed forms inside a RunE body:
//   - state.out() / cmd.OutOrStdout() / cmd.OutOrStderr()
//   - fmt.Fprint*(state.out(), ...) and fmt.Fprint*(os.Stderr, ...)
//   - printData(state, ...) and printQuietIDs(state, ...)
//
// Forbidden forms (without an explicit `// printdata-ok:` exemption comment
// on the same line):
//   - os.Stdout used as an io.Writer
//   - fmt.Print, fmt.Println, fmt.Printf called directly (no Writer arg)
//
// To suppress a flagged line, add `// printdata-ok: <reason>` at the end of
// the line. The reason must explain why the bypass is intentional (e.g.
// streaming subprocess output, raw-byte passthrough that callers pipe).
func TestRunEHandlersDoNotBypassPrintData(t *testing.T) {
	t.Parallel()

	dir := "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read cmd/strait: %v", err)
	}

	fset := token.NewFileSet()
	type violation struct {
		file string
		line int
		text string
	}
	var violations []violation

	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		// Index `// printdata-ok:` comments by line number so we can suppress
		// flagged lines on the same line as the comment.
		exempt := map[int]bool{}
		for _, cg := range file.Comments {
			for _, c := range cg.List {
				if strings.Contains(c.Text, "printdata-ok") {
					exempt[fset.Position(c.Slash).Line] = true
				}
			}
		}

		// Find every cobra Command literal (composite literal with an `RunE`
		// key) and inspect the function body bound to RunE.
		ast.Inspect(file, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok || key.Name != "RunE" {
					continue
				}
				fn, ok := kv.Value.(*ast.FuncLit)
				if !ok || fn.Body == nil {
					continue
				}
				ast.Inspect(fn.Body, func(inner ast.Node) bool {
					call, ok := inner.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					pkg, ok := sel.X.(*ast.Ident)
					if !ok {
						return true
					}
					line := fset.Position(call.Pos()).Line
					if exempt[line] {
						return true
					}
					if pkg.Name == "fmt" {
						switch sel.Sel.Name {
						case "Print", "Println", "Printf":
							violations = append(violations, violation{
								file: name, line: line,
								text: "fmt." + sel.Sel.Name + " bypasses state.out() / printData",
							})
						}
					}
					return true
				})
				// Separately scan for os.Stdout used as an expression
				// (e.g. passed as a writer arg). Excludes `os.Stdout.Write`
				// and `os.Stdout.Stat` since those are method calls — but
				// the .Write case is still a bypass and we want to flag it.
				ast.Inspect(fn.Body, func(inner ast.Node) bool {
					sel, ok := inner.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					pkg, ok := sel.X.(*ast.Ident)
					if !ok {
						return true
					}
					if pkg.Name != "os" || sel.Sel.Name != "Stdout" {
						return true
					}
					line := fset.Position(sel.Pos()).Line
					if exempt[line] {
						return true
					}
					violations = append(violations, violation{
						file: name, line: line,
						text: "os.Stdout used directly — route through state.out()",
					})
					return true
				})
			}
			return true
		})
	}

	if len(violations) > 0 {
		var b strings.Builder
		b.WriteString("RunE handlers must not bypass printData / state.out():\n")
		for _, v := range violations {
			b.WriteString("  ")
			b.WriteString(v.file)
			b.WriteString(":")
			b.WriteString(itoa(v.line))
			b.WriteString(": ")
			b.WriteString(v.text)
			b.WriteString("\n")
		}
		b.WriteString("\nIf the bypass is intentional, add `// printdata-ok: <reason>` to the line.\n")
		t.Fatal(b.String())
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
