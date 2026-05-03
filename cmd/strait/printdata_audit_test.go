package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// printdataExempt matches a properly-formatted exemption comment. The colon
// and a non-empty reason are both required, so accidental embeddings like
// `// printdata-okay-not-allowed` do not silence the audit.
var printdataExempt = regexp.MustCompile(`^\s*//\s*printdata-ok:\s+\S+`)

// TestRunEHandlersDoNotBypassPrintData walks every function declaration and
// function literal in cmd/strait/*.go (production code only) and asserts that
// none of them write primary output by reaching past printData / state.out()
// to the global os.Stdout (or to fmt.Print/Println/Printf, which write to
// os.Stdout implicitly), or via the print/println builtins.
//
// Why every function, not just RunE bodies: a handler that delegates to a
// helper (renderDAG, formatTable, etc.) would otherwise smuggle a bypass past
// the audit. The whole package is in scope so helpers, PreRunE, named-function
// RunE assignments (cmd.RunE = handleX), and PostRunE are all covered.
//
// The motivating regression: `workflows visualize` originally used
// fmt.Print(rendered) instead of routing through state.out(), so --format
// json/yaml were silently ignored. This test catches the whole class.
//
// Allowed forms anywhere in cmd/strait/*.go:
//   - state.out() / cmd.OutOrStdout() / cmd.OutOrStderr()
//   - fmt.Fprint*(state.out(), ...) and fmt.Fprint*(os.Stderr, ...)
//   - printData(state, ...) and printQuietIDs(state, ...)
//
// Forbidden forms (without an explicit `// printdata-ok: <reason>` exemption
// comment on the same line):
//   - os.Stdout used as an io.Writer
//   - fmt.Print, fmt.Println, fmt.Printf called directly (no Writer arg)
//   - print(...) / println(...) builtins
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

		// Index `// printdata-ok: <reason>` exemption comments by line. The
		// regex requires a colon and a non-empty reason so accidental
		// substring matches do not silence violations.
		exempt := map[int]bool{}
		for _, cg := range file.Comments {
			for _, c := range cg.List {
				if printdataExempt.MatchString(c.Text) {
					exempt[fset.Position(c.Slash).Line] = true
				}
			}
		}

		// Walk every function declaration and every function literal in the
		// file. This deliberately covers more than just cobra RunE — helpers
		// called from RunE bodies, PreRunE, and named-function handlers all
		// route their output the same way and should obey the same rule.
		ast.Inspect(file, func(n ast.Node) bool {
			var body *ast.BlockStmt
			switch fn := n.(type) {
			case *ast.FuncDecl:
				body = fn.Body
			case *ast.FuncLit:
				body = fn.Body
			default:
				return true
			}
			if body == nil {
				return true
			}
			ast.Inspect(body, func(inner ast.Node) bool {
				switch x := inner.(type) {
				case *ast.CallExpr:
					line := fset.Position(x.Pos()).Line
					if exempt[line] {
						return true
					}
					// fmt.Print / Println / Printf — package-qualified calls.
					if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
						if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "fmt" {
							switch sel.Sel.Name {
							case "Print", "Println", "Printf":
								violations = append(violations, violation{
									file: name, line: line,
									text: "fmt." + sel.Sel.Name + " bypasses state.out() / printData",
								})
							}
						}
					}
					// print / println builtins — bare identifier calls.
					if id, ok := x.Fun.(*ast.Ident); ok {
						switch id.Name {
						case "print", "println":
							violations = append(violations, violation{
								file: name, line: line,
								text: id.Name + " builtin bypasses state.out() / printData",
							})
						}
					}
				case *ast.SelectorExpr:
					pkg, ok := x.X.(*ast.Ident)
					if !ok {
						return true
					}
					if pkg.Name != "os" || x.Sel.Name != "Stdout" {
						return true
					}
					line := fset.Position(x.Pos()).Line
					if exempt[line] {
						return true
					}
					violations = append(violations, violation{
						file: name, line: line,
						text: "os.Stdout used directly — route through state.out()",
					})
				}
				return true
			})
			return true
		})
	}

	if len(violations) > 0 {
		var b strings.Builder
		b.WriteString("Output must route through printData(state, ...) or state.out():\n")
		for _, v := range violations {
			b.WriteString("  ")
			b.WriteString(v.file)
			b.WriteString(":")
			b.WriteString(strconv.Itoa(v.line))
			b.WriteString(": ")
			b.WriteString(v.text)
			b.WriteString("\n")
		}
		b.WriteString("\nPrefer routing output through printData(state, ...) or fmt.Fprint*(state.out(), ...).\n")
		b.WriteString("If the bypass is intentional, add `// printdata-ok: <reason>` to the line.\n")
		t.Fatal(b.String())
	}
}
