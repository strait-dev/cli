package client

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestEndpointsMatchOpenAPISpec is the contract guard against CLI/server path
// drift. It statically extracts every REST endpoint the client calls (from the
// do*JSON helpers, plus a small set of hand-built streaming requests) and
// asserts each one exists in the server's vendored OpenAPI spec.
//
// Refresh the spec with `make refresh-openapi` (pulls /reference/openapi.json
// from a running server). If this test fails, the CLI is calling a path the
// server does not expose — fix the path in api.go, do not silence the test.
func TestEndpointsMatchOpenAPISpec(t *testing.T) {
	t.Parallel()

	specSet := loadSpecEndpoints(t)
	cliEndpoints := extractClientEndpoints(t)

	if len(cliEndpoints) == 0 {
		t.Fatal("no client endpoints extracted; the AST walker is broken")
	}

	var missing []string
	for _, ep := range cliEndpoints {
		key := ep.method + " " + ep.path
		if _, ok := specSet[key]; !ok {
			missing = append(missing, key+"  (from "+ep.source+")")
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("%d CLI endpoint(s) not found in OpenAPI spec (path drift):\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
}

type cliEndpoint struct {
	method string
	path   string // normalized: path params collapsed to "{}"
	source string // file:line for diagnostics
}

var pathParamRe = regexp.MustCompile(`\{[^}]+\}`)

// normalizePath collapses any {param} placeholder to "{}" and squashes repeated
// slashes so spec paths and CLI paths compare equal regardless of param naming.
func normalizePath(p string) string {
	p = pathParamRe.ReplaceAllString(p, "{}")
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	return strings.TrimRight(p, "/")
}

func loadSpecEndpoints(t *testing.T) map[string]struct{} {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "openapi.json"))
	if err != nil {
		t.Fatalf("read vendored spec: %v (run `make refresh-openapi`)", err)
	}
	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	set := make(map[string]struct{})
	for p, methods := range spec.Paths {
		for m := range methods {
			mu := strings.ToUpper(m)
			switch mu {
			case "GET", "POST", "PUT", "PATCH", "DELETE":
				set[mu+" "+normalizePath(p)] = struct{}{}
			}
		}
	}
	return set
}

// httpMethodConst maps an http.MethodX selector to its verb.
var httpMethodConst = map[string]string{
	"MethodGet": "GET", "MethodPost": "POST", "MethodPut": "PUT",
	"MethodPatch": "PATCH", "MethodDelete": "DELETE",
}

func extractClientEndpoints(t *testing.T) []cliEndpoint {
	t.Helper()
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	var endpoints []cliEndpoint
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		node, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			var methodArg, pathArg ast.Expr
			method := ""
			switch sel.Sel.Name {
			case "doJSON", "doJSONWithHeaders":
				if len(call.Args) < 3 {
					return true
				}
				methodArg, pathArg = call.Args[1], call.Args[2]
			case "doListJSON", "doListAllJSON":
				if len(call.Args) < 2 {
					return true
				}
				method, pathArg = "GET", call.Args[1]
			default:
				return true
			}

			if method == "" {
				method = resolveMethod(methodArg)
				if method == "" {
					return true // dynamic method; skip
				}
			}
			p, ok := resolvePath(pathArg)
			if !ok {
				return true // dynamic/unresolvable path; skip
			}
			pos := fset.Position(call.Pos())
			endpoints = append(endpoints, cliEndpoint{
				method: method,
				path:   normalizePath(p),
				source: filepath.Base(pos.Filename) + ":" + strconv.Itoa(pos.Line),
			})
			return true
		})
	}

	// Hand-built streaming requests not routed through the do* helpers.
	endpoints = append(endpoints,
		cliEndpoint{method: "GET", path: "/v1/runs/{}/stream", source: "stream.go (manual)"},
	)
	return endpoints
}

func resolveMethod(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.SelectorExpr:
		return httpMethodConst[v.Sel.Name]
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			s, _ := strconv.Unquote(v.Value)
			return strings.ToUpper(s)
		}
	}
	return ""
}

// resolvePath reconstructs a path from a string literal or a path.Join(...)
// call. Non-literal segments (identifiers, function calls) become "{}".
func resolvePath(e ast.Expr) (string, bool) {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			s, err := strconv.Unquote(v.Value)
			if err != nil {
				return "", false
			}
			return s, true
		}
	case *ast.CallExpr:
		sel, ok := v.Fun.(*ast.SelectorExpr)
		if !ok {
			return "", false
		}
		pkg, _ := sel.X.(*ast.Ident)
		isJoin := (pkg != nil && pkg.Name == "path" && sel.Sel.Name == "Join") ||
			sel.Sel.Name == "joinPath"
		if !isJoin {
			return "", false
		}
		parts := make([]string, 0, len(v.Args))
		for i, arg := range v.Args {
			if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				s, err := strconv.Unquote(lit.Value)
				if err != nil {
					return "", false
				}
				parts = append(parts, s)
				continue
			}
			// First arg must be a literal prefix to anchor the path.
			if i == 0 {
				return "", false
			}
			parts = append(parts, "{}")
		}
		return strings.Join(parts, "/"), true
	}
	return "", false
}
