package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/wizard"
)

//go:embed all:templates
var initTemplates embed.FS

// projectNamePlaceholder is the literal token replaced with the user-supplied
// project name in every rendered template file. We use a fixed sentinel rather
// than text/template syntax so the embedded sources are valid Go/TypeScript on
// disk and editors can lint them as-is.
const projectNamePlaceholder = "STRAIT_TEMPLATE_PROJECT_NAME"

// initTemplateNames returns the list of available template directories under
// cmd/strait/templates, sorted alphabetically. The list is computed at runtime
// from the embedded FS so adding a new template directory is a single change.
func initTemplateNames() ([]string, error) {
	entries, err := fs.ReadDir(initTemplates, "templates")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func newInitCommand(state *appState) *cobra.Command {
	var template string
	var projectName string
	var force bool
	var listTemplates bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold a new Strait project from a template",
		Long: "Scaffold a new Strait project from one of the bundled templates. " +
			"Each template is a working starter project that wires the Strait SDK " +
			"into a specific runtime (Vercel, Cloudflare Workers, AWS Lambda, etc.).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			names, err := initTemplateNames()
			if err != nil {
				return fmt.Errorf("read embedded templates: %w", err)
			}

			if listTemplates {
				w := state.out()
				for _, name := range names {
					fmt.Fprintln(w, name)
				}
				return nil
			}

			if strings.TrimSpace(template) == "" {
				return fmt.Errorf("--template is required (one of: %s)", strings.Join(names, ", "))
			}

			templateDir := filepath.ToSlash(filepath.Join("templates", template))
			if _, err := fs.Stat(initTemplates, templateDir); err != nil {
				return fmt.Errorf("unknown template %q (available: %s)", template, strings.Join(names, ", "))
			}

			if projectName == "" {
				projectName = template
			}
			if err := wizard.ValidateProjectName(projectName); err != nil {
				return fmt.Errorf("invalid --name: %w", err)
			}

			destDir, err := filepath.Abs(projectName)
			if err != nil {
				return fmt.Errorf("resolve destination: %w", err)
			}

			if err := scaffoldTemplate(initTemplates, templateDir, destDir, projectName, force); err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(cmd.ErrOrStderr(), styles.Success("Scaffolded "+styles.Bold.Render(projectName)+" from template "+template))
				fmt.Fprintln(cmd.ErrOrStderr(), styles.KeyValue("Path", destDir))
				fmt.Fprintln(cmd.ErrOrStderr(), styles.KeyValue("Next", "see "+filepath.Join(projectName, "README.md")+" for setup steps"))
				return nil
			}
			return printData(state, map[string]any{
				"template": template,
				"name":     projectName,
				"path":     destDir,
				"created":  true,
			})
		},
	}

	cmd.Flags().StringVar(&template, "template", "", "template to scaffold (use --list to see available templates)")
	cmd.Flags().StringVar(&projectName, "name", "", "project name (also used as destination directory; defaults to the template name)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing files in the destination directory")
	cmd.Flags().BoolVar(&listTemplates, "list", false, "list available templates and exit")

	_ = cmd.RegisterFlagCompletionFunc("template", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		names, err := initTemplateNames()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

// scaffoldTemplate copies every file under sourceRoot in the embedded FS to
// destRoot, replacing the project-name placeholder in file contents and
// honoring force-overwrite semantics. Returns an error on the first conflict
// when force is false; partial extractions on failure are left in place so the
// user can inspect what landed.
func scaffoldTemplate(src fs.FS, sourceRoot, destRoot, projectName string, force bool) error {
	if !force {
		if _, err := os.Stat(destRoot); err == nil {
			entries, readErr := os.ReadDir(destRoot)
			if readErr == nil && len(entries) > 0 {
				return fmt.Errorf("destination %q already exists and is not empty (pass --force to overwrite)", destRoot)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat destination: %w", err)
		}
	}

	if err := os.MkdirAll(destRoot, 0o750); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}

	return fs.WalkDir(src, sourceRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		target := filepath.Join(destRoot, rel)
		// Templates ship `gitignore` files that we rename to `.gitignore` on
		// extraction — Go's embed directive excludes dotfiles by default.
		if filepath.Base(target) == "gitignore" {
			target = filepath.Join(filepath.Dir(target), ".gitignore")
		}
		// `.tmpl` suffixes hide embedded Go sources (go.mod, main.go) from the
		// parent module's `go build ./...` traversal; the scaffolder restores
		// the canonical name in the user's project.
		if trimmed, ok := strings.CutSuffix(target, ".tmpl"); ok {
			target = trimmed
		}

		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}

		raw, readErr := fs.ReadFile(src, path)
		if readErr != nil {
			return fmt.Errorf("read template file %q: %w", path, readErr)
		}
		rendered := strings.ReplaceAll(string(raw), projectNamePlaceholder, projectName)

		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return fmt.Errorf("create parent dir for %q: %w", target, err)
		}
		if err := os.WriteFile(target, []byte(rendered), 0o600); err != nil {
			return fmt.Errorf("write %q: %w", target, err)
		}
		return nil
	})
}
