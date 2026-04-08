package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/wizard"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

type straitConfigJSON struct {
	Project  projectBlock    `json:"project"`
	Runtime  string          `json:"runtime,omitempty"`
	Jobs     []jobBlock      `json:"jobs,omitempty"`
	Workflow []workflowBlock `json:"workflows,omitempty"`
}

type projectBlock struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type jobBlock struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	EndpointURL string `json:"endpointUrl,omitempty"`
	Cron        string `json:"cron,omitempty"`
}

type workflowBlock struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type initConfigFile struct {
	Server        string `yaml:"server"`
	Project       string `yaml:"project"`
	Format        string `yaml:"format"`
	ActiveContext string `yaml:"active_context"`
}

type initJobManifest struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata"`
	Spec       map[string]any    `yaml:"spec"`
}

type initWorkflowManifest struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata"`
	Spec       map[string]any    `yaml:"spec"`
}

func newInitCommand(state *appState) *cobra.Command {
	var (
		yes         bool
		force       bool
		template    string
		name        string
		runtime     string
		withJob     bool
		jobName     string
		jobEndpoint string
		jobCron     string
		fromServer  bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new strait project",
		Long: `Initialize a new strait project with configuration files.

In interactive mode (default when TTY), a wizard guides you through setup.
In non-interactive mode (--yes), all values come from flags.`,
		Example: `  strait init
  strait init --yes --name my-api --runtime typescript
  strait init --yes --name my-api --runtime go --with-job --job-name process-payment --job-endpoint http://localhost:3000/jobs/payment
  strait init --template full --name demo
  strait init --from-server --project proj-1`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// --from-server: scaffold manifests from live server jobs
			if fromServer {
				return runInitFromServer(cmd, state, force)
			}

			// Interactive mode: TTY + no --yes flag
			if !yes && stdoutIsTTY() {
				result, err := wizard.RunInitWizard()
				if err != nil {
					return err
				}
				name = result.ProjectName
				runtime = result.Runtime
				withJob = result.WithJob
				jobName = result.JobName
				jobEndpoint = result.JobEndpoint
				jobCron = result.JobCron
			} else if !yes {
				if state.opts.nonInteractive {
					return fmt.Errorf("non-interactive mode: use --yes with flags for non-interactive init")
				}
				return fmt.Errorf("interactive mode requires a TTY; use --yes with flags for non-interactive init")
			}

			// Validate inputs
			name = strings.TrimSpace(name)
			if err := wizard.ValidateProjectName(name); err != nil {
				return err
			}
			if runtime != "" {
				if err := wizard.ValidateRuntime(runtime); err != nil {
					return err
				}
			}

			// Check for existing config (unless --force)
			configPath := "strait.config.json"
			if !force {
				if _, err := os.Stat(configPath); err == nil {
					return fmt.Errorf("config file %s already exists (use --force to overwrite)", configPath)
				}
			}

			// Write strait.config.json
			cfg := straitConfigJSON{
				Project: projectBlock{ID: name, Name: name},
				Runtime: runtime,
			}
			if withJob && jobName != "" {
				slug := wizard.GenerateSlug(jobName)
				cfg.Jobs = append(cfg.Jobs, jobBlock{
					Slug:        slug,
					Name:        jobName,
					EndpointURL: jobEndpoint,
					Cron:        jobCron,
				})
			}

			encoded, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return fmt.Errorf("encode config: %w", err)
			}
			if err := os.WriteFile(configPath, append(encoded, '\n'), 0o600); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			// Write .strait.yaml (local CLI config)
			configStatus, err := writeInitConfig(name)
			if err != nil {
				return fmt.Errorf("writing CLI config: %w", err)
			}

			// Update .gitignore
			gitignoreStatus := updateGitignore()

			// Write .straitignore
			straitignoreStatus, straitignoreErr := writeStraitIgnore(runtime)
			if straitignoreErr != nil {
				return fmt.Errorf("writing .straitignore: %w", straitignoreErr)
			}

			// Write declarative definitions (legacy template mode)
			if template == "full" || template == "minimal" {
				envStatus, envErr := writeInitEnv()
				if envErr != nil {
					return fmt.Errorf("writing .env: %w", envErr)
				}
				dcStatus, dcErr := writeInitDockerCompose()
				if dcErr != nil {
					return fmt.Errorf("writing docker-compose: %w", dcErr)
				}
				manifestStatus, mErr := writeInitJobManifest(template, name)
				if mErr != nil {
					return fmt.Errorf("writing job manifest: %w", mErr)
				}
				wfStatus, wfErr := writeInitWorkflowManifest(template, name)
				if wfErr != nil {
					return fmt.Errorf("writing workflow manifest: %w", wfErr)
				}

				return printData(state, map[string]any{
					"project":  name,
					"runtime":  runtime,
					"template": template,
					"files": []map[string]any{
						{"path": configPath, "status": "created"},
						{"path": ".strait.yaml", "status": configStatus},
						{"path": ".gitignore", "status": gitignoreStatus},
						{"path": ".straitignore", "status": straitignoreStatus},
						{"path": ".env", "status": envStatus},
						{"path": "docker-compose.yml", "status": dcStatus},
						{"path": "definitions/jobs.yaml", "status": manifestStatus},
						{"path": "definitions/workflows.yaml", "status": wfStatus},
					},
				})
			}

			files := []map[string]any{
				{"path": configPath, "status": "created"},
				{"path": ".strait.yaml", "status": configStatus},
				{"path": ".gitignore", "status": gitignoreStatus},
				{"path": ".straitignore", "status": straitignoreStatus},
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Initialized project "+styles.Bold.Render(name)))
				for _, f := range files {
					fmt.Fprintln(os.Stderr, styles.KeyValue(f["status"].(string), styles.FilePath(f["path"].(string))))
				}
				return nil
			}

			return printData(state, map[string]any{
				"project": name,
				"runtime": runtime,
				"files":   files,
			})
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "run non-interactive initialization with flags")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing config files")
	cmd.Flags().StringVar(&template, "template", "", "template mode (minimal|full) for legacy definitions")
	cmd.Flags().StringVar(&name, "name", "", "project name")
	cmd.Flags().StringVar(&runtime, "runtime", "", "project runtime (go, python, typescript, ruby, rust, node, bun, docker)")
	cmd.Flags().BoolVar(&withJob, "with-job", false, "include a starter job")
	cmd.Flags().StringVar(&jobName, "job-name", "", "starter job name (requires --with-job)")
	cmd.Flags().StringVar(&jobEndpoint, "job-endpoint", "", "starter job endpoint URL (requires --with-job)")
	cmd.Flags().StringVar(&jobCron, "job-cron", "", "starter job cron schedule (optional)")
	cmd.Flags().BoolVar(&fromServer, "from-server", false, "scaffold manifests by fetching existing jobs and workflows from the server (requires --project or STRAIT_PROJECT)")

	return cmd
}

func updateGitignore() string {
	const entry = ".strait/"
	path := ".gitignore"

	content, err := os.ReadFile(path)
	if err != nil {
		// No .gitignore — create one
		if err := os.WriteFile(path, []byte(entry+"\n"), 0o600); err != nil {
			return "error"
		}
		return "created"
	}

	// Check if already present
	for line := range strings.SplitSeq(string(content), "\n") {
		if strings.TrimSpace(line) == entry {
			return "skipped"
		}
	}

	// Append
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}
	content = append(content, []byte(entry+"\n")...)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return "error"
	}
	return "updated"
}

func writeInitConfig(projectName string) (string, error) {
	path := ".strait.yaml"
	if _, err := os.Stat(path); err == nil {
		return "skipped", nil
	}

	cfg := initConfigFile{
		Server:        "http://localhost:8080",
		Project:       projectName,
		Format:        "table",
		ActiveContext: "default",
	}

	encoded, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return "", err
	}

	return "created", nil
}

func writeInitEnv() (string, error) {
	if _, err := os.Stat(".env"); err == nil {
		return "skipped", nil
	}

	if _, err := os.Stat(".env.example"); err == nil {
		content, readErr := os.ReadFile(".env.example")
		if readErr != nil {
			return "", readErr
		}
		if err := os.WriteFile(".env", content, 0o600); err != nil {
			return "", err
		}
		return "created", nil
	}

	defaultEnv := []byte("DATABASE_URL=postgres://strait:strait@localhost:5432/strait?sslmode=disable\nREDIS_URL=redis://localhost:6379\n")
	if err := os.WriteFile(".env", defaultEnv, 0o600); err != nil {
		return "", err
	}

	return "created", nil
}

func writeInitDockerCompose() (string, error) {
	path := "docker-compose.yml"
	if _, err := os.Stat(path); err == nil {
		return "skipped", nil
	}

	compose := []byte("services:\n  postgres:\n    image: postgres:16\n    environment:\n      POSTGRES_USER: strait\n      POSTGRES_PASSWORD: strait\n      POSTGRES_DB: strait\n    ports:\n      - \"5432:5432\"\n\n  redis:\n    image: redis:7\n    ports:\n      - \"6379:6379\"\n")
	if err := os.WriteFile(path, compose, 0o600); err != nil {
		return "", err
	}

	return "created", nil
}

func writeInitJobManifest(template, projectName string) (string, error) {
	dir := "definitions"
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}

	target := filepath.Join(dir, "jobs.yaml")
	if _, err := os.Stat(target); err == nil {
		return "skipped", nil
	}

	jobName := "example-job"
	jobSlug := "example-job"
	jobDescription := "example job definition"
	jobCron := ""
	if template == "full" {
		jobName = "example-full-job"
		jobSlug = "example-full-job"
		jobDescription = "example full template job"
		jobCron = "*/5 * * * *"
	}

	m := initJobManifest{
		APIVersion: "v1",
		Kind:       "Job",
		Metadata:   map[string]string{"name": jobName},
		Spec: map[string]any{
			"project_id":   projectName,
			"slug":         jobSlug,
			"description":  jobDescription,
			"cron":         jobCron,
			"endpoint_url": "http://localhost:3000/webhook",
			"timeout_secs": 60,
			"max_attempts": 3,
		},
	}

	encoded, err := yaml.Marshal(m)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(target, encoded, 0o600); err != nil {
		return "", err
	}

	return "created", nil
}

func writeInitWorkflowManifest(template, projectName string) (string, error) {
	if template != "full" {
		return "not_applicable", nil
	}

	dir := "definitions"
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}

	target := filepath.Join(dir, "workflows.yaml")
	if _, err := os.Stat(target); err == nil {
		return "skipped", nil
	}

	m := initWorkflowManifest{
		APIVersion: "v1",
		Kind:       "Workflow",
		Metadata:   map[string]string{"name": "example-full-workflow"},
		Spec: map[string]any{
			"project_id":  projectName,
			"slug":        "example-full-workflow",
			"description": "example full template workflow",
			"enabled":     true,
			"steps": []map[string]any{
				{
					"step_ref":   "send_webhook",
					"job_id":     "example-full-job",
					"depends_on": []string{},
					"on_failure": "fail",
				},
			},
		},
	}

	encoded, err := yaml.Marshal(m)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(target, encoded, 0o600); err != nil {
		return "", err
	}

	return "created", nil
}

// writeStraitIgnore writes a .straitignore file with common and runtime-specific
// patterns. Returns "skipped" if the file already exists.
func writeStraitIgnore(runtime string) (string, error) {
	path := ".straitignore"
	if _, err := os.Stat(path); err == nil {
		return "skipped", nil
	}

	lines := []string{
		"# Common — always excluded from source packs",
		".git/",
		".DS_Store",
		"*.log",
		"*.tmp",
		".env",
		".env.*",
		"",
		"# Build outputs",
		"dist/",
		"build/",
		"out/",
		"tmp/",
		".tmp/",
		"",
		"# Secrets and certificates",
		"*.pem",
		"*.key",
		"*.crt",
		"*.p12",
	}

	// Normalise aliases to canonical names.
	normalised := runtime
	switch runtime {
	case "node", "bun", "js":
		normalised = "typescript"
	}

	switch normalised {
	case "typescript":
		lines = append(lines,
			"",
			"# Node.js / TypeScript",
			"node_modules/",
			".next/",
			".nuxt/",
			".turbo/",
			"coverage/",
			"*.tsbuildinfo",
			".cache/",
		)
	case "python":
		lines = append(lines,
			"",
			"# Python",
			"__pycache__/",
			"*.pyc",
			"*.pyo",
			".venv/",
			"venv/",
			".pytest_cache/",
			"*.egg-info/",
			".mypy_cache/",
		)
	case "go":
		lines = append(lines,
			"",
			"# Go",
			"vendor/",
			"*.test",
			"*.prof",
		)
	case "rust":
		lines = append(lines,
			"",
			"# Rust",
			"target/",
		)
	case "ruby":
		lines = append(lines,
			"",
			"# Ruby",
			".bundle/",
			"vendor/bundle/",
			"tmp/",
			"log/",
			"*.gem",
		)
	}

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return "created", nil
}

// runInitFromServer fetches jobs and workflows from the server and scaffolds
// YAML manifest files (definitions/jobs.yaml, definitions/workflows.yaml).
func runInitFromServer(cmd *cobra.Command, state *appState, force bool) error {
	projectID := state.opts.projectID
	if projectID == "" {
		return fmt.Errorf("--project (or STRAIT_PROJECT) is required for --from-server")
	}

	cli, err := newAPIClient(state)
	if err != nil {
		return err
	}

	jobs, err := cli.ListJobs(cmd.Context(), projectID)
	if err != nil {
		return fmt.Errorf("fetch jobs: %w", err)
	}

	workflows, err := cli.ListWorkflows(cmd.Context(), projectID)
	if err != nil {
		return fmt.Errorf("fetch workflows: %w", err)
	}

	if len(jobs) == 0 && len(workflows) == 0 {
		if isTTYRich(state) {
			fmt.Fprintln(os.Stderr, styles.Warn("no jobs or workflows found in project "+projectID))
		}
		return printData(state, map[string]any{
			"project_id": projectID,
			"jobs":       0,
			"workflows":  0,
			"files":      []string{},
		})
	}

	if err := os.MkdirAll("definitions", 0o750); err != nil {
		return fmt.Errorf("create definitions directory: %w", err)
	}

	var written []string

	// Scaffold jobs manifest
	if len(jobs) > 0 {
		jobsPath := filepath.Join("definitions", "jobs.yaml")
		if !force {
			if _, statErr := os.Stat(jobsPath); statErr == nil {
				return fmt.Errorf("%s already exists (use --force to overwrite)", jobsPath)
			}
		}

		var buf strings.Builder
		for i, job := range jobs {
			if i > 0 {
				buf.WriteString("---\n")
			}
			m := initJobManifest{
				APIVersion: "strait.dev/v1",
				Kind:       "Job",
				Metadata:   map[string]string{"name": job.Name},
				Spec: map[string]any{
					"slug":         job.Slug,
					"project_id":   projectID,
					"endpoint_url": job.EndpointURL,
					"timeout_secs": job.TimeoutSecs,
					"max_attempts": job.MaxAttempts,
				},
			}
			if job.Cron != "" {
				m.Spec["cron"] = job.Cron
			}
			out, marshalErr := yaml.Marshal(m)
			if marshalErr != nil {
				return fmt.Errorf("encode job manifest: %w", marshalErr)
			}
			buf.Write(out)
		}
		if err := os.WriteFile(jobsPath, []byte(buf.String()), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", jobsPath, err)
		}
		written = append(written, jobsPath)
	}

	// Scaffold workflows manifest
	if len(workflows) > 0 {
		wfPath := filepath.Join("definitions", "workflows.yaml")
		if !force {
			if _, statErr := os.Stat(wfPath); statErr == nil {
				return fmt.Errorf("%s already exists (use --force to overwrite)", wfPath)
			}
		}

		var buf strings.Builder
		for i, wf := range workflows {
			if i > 0 {
				buf.WriteString("---\n")
			}
			m := initWorkflowManifest{
				APIVersion: "strait.dev/v1",
				Kind:       "Workflow",
				Metadata:   map[string]string{"name": wf.Name},
				Spec: map[string]any{
					"slug":       wf.Slug,
					"project_id": projectID,
				},
			}
			out, marshalErr := yaml.Marshal(m)
			if marshalErr != nil {
				return fmt.Errorf("encode workflow manifest: %w", marshalErr)
			}
			buf.Write(out)
		}
		if err := os.WriteFile(wfPath, []byte(buf.String()), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", wfPath, err)
		}
		written = append(written, wfPath)
	}

	if isTTYRich(state) {
		fmt.Fprintln(os.Stderr, styles.Success(fmt.Sprintf("Scaffolded %d job(s), %d workflow(s) from server", len(jobs), len(workflows))))
		for _, f := range written {
			fmt.Fprintln(os.Stderr, styles.KeyValue("created", styles.FilePath(f)))
		}
		return nil
	}
	return printData(state, map[string]any{
		"project_id": projectID,
		"jobs":       len(jobs),
		"workflows":  len(workflows),
		"files":      written,
	})
}
