package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestInitTemplateNames_IncludesAllShippedTemplates(t *testing.T) {
	t.Parallel()

	got, err := initTemplateNames()
	if err != nil {
		t.Fatalf("initTemplateNames: %v", err)
	}

	want := []string{
		"cloudflare",
		"express",
		"go-chi-serve",
		"go-worker",
		"k8s-worker",
		"lambda",
		"netlify",
		"vercel",
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("templates = %v, want %v", got, want)
	}
}

func TestInitCommand_RequiresTemplate(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --template missing")
	}
	if !strings.Contains(err.Error(), "--template is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitCommand_RejectsUnknownTemplate(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--template", "nope"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
	if !strings.Contains(err.Error(), "unknown template") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitCommand_ListFlag(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	out := &bytes.Buffer{}
	state.stdout = out
	cmd := newInitCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	for _, want := range []string{"vercel", "cloudflare", "go-worker"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("--list output missing %q: %s", want, out.String())
		}
	}
}

func TestInitCommand_ScaffoldsVercelTemplate(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp) // t.Chdir is incompatible with t.Parallel; run serially.

	state := &appState{opts: &rootOptions{}}
	state.stdout = &bytes.Buffer{}
	cmd := newInitCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--template", "vercel", "--name", "demo-app"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	dest := filepath.Join(tmp, "demo-app")
	pkg, err := os.ReadFile(filepath.Join(dest, "package.json"))
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}
	if !strings.Contains(string(pkg), `"name": "demo-app"`) {
		t.Fatalf("placeholder not substituted in package.json: %s", pkg)
	}

	for _, rel := range []string{
		"src/jobs.ts",
		"app/api/strait/route.ts",
		"strait.json",
		"README.md",
		".gitignore",
	} {
		if _, err := os.Stat(filepath.Join(dest, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
	rawConfig, err := os.ReadFile(filepath.Join(dest, "strait.json"))
	if err != nil {
		t.Fatalf("read strait.json: %v", err)
	}
	if !strings.Contains(string(rawConfig), `"$schema": "https://schemas.strait.dev/v1/strait.json"`) {
		t.Fatalf("strait.json missing $schema: %s", rawConfig)
	}
}

func TestInitCommand_ScaffoldsGoTemplateRenamesTmpl(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp) // t.Chdir is incompatible with t.Parallel; run serially.

	state := &appState{opts: &rootOptions{}}
	state.stdout = &bytes.Buffer{}
	cmd := newInitCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--template", "go-worker", "--name", "my-worker"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	dest := filepath.Join(tmp, "my-worker")
	for _, rel := range []string{"go.mod", "main.go", "README.md", "strait.json"} {
		if _, err := os.Stat(filepath.Join(dest, rel)); err != nil {
			t.Fatalf("missing %s after .tmpl rename: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "main.go.tmpl")); !os.IsNotExist(err) {
		t.Fatalf("main.go.tmpl should not exist in scaffolded output (err=%v)", err)
	}

	mod, err := os.ReadFile(filepath.Join(dest, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if !strings.Contains(string(mod), "module my-worker") {
		t.Fatalf("placeholder not substituted in go.mod: %s", mod)
	}
}

func TestInitCommand_RefusesNonEmptyDestinationWithoutForce(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp) // t.Chdir is incompatible with t.Parallel; run serially.

	dest := filepath.Join(tmp, "occupied")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "preexisting.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	state := &appState{opts: &rootOptions{}}
	state.stdout = &bytes.Buffer{}
	cmd := newInitCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--template", "express", "--name", "occupied"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when destination is non-empty")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitCommand_RejectsInvalidProjectName(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp) // t.Chdir is incompatible with t.Parallel; run serially.

	state := &appState{opts: &rootOptions{}}
	state.stdout = &bytes.Buffer{}
	cmd := newInitCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--template", "vercel", "--name", "Bad Name"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid project name")
	}
	if !strings.Contains(err.Error(), "invalid --name") {
		t.Fatalf("unexpected error: %v", err)
	}
}
