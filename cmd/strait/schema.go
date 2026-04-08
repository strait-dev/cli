package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// schemaField describes one field in a resource schema.
type schemaField struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// schemaResource is the top-level schema descriptor returned by `strait schema <resource>`.
type schemaResource struct {
	Resource    string        `json:"resource"`
	Description string        `json:"description"`
	Fields      []schemaField `json:"fields"`
}

func newSchemaCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Print machine-readable schemas for Strait resources",
		Long: `Print field schemas for Strait resources as JSON.
Useful for agents and scripts that need to know valid field names,
types, and enum values without hitting the API.`,
		Example: `  strait schema runtimes
  strait schema job
  strait schema deployment
  strait schema workflow
  strait schema run`,
	}

	cmd.AddCommand(newSchemaRuntimesCommand(state))
	cmd.AddCommand(newSchemaJobCommand(state))
	cmd.AddCommand(newSchemaDeploymentCommand(state))
	cmd.AddCommand(newSchemaWorkflowCommand(state))
	cmd.AddCommand(newSchemaRunCommand(state))

	return cmd
}

func newSchemaRuntimesCommand(_ *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "runtimes",
		Short: "List supported code-first deployment runtimes",
		RunE: func(_ *cobra.Command, _ []string) error {
			runtimes := []map[string]string{
				{"runtime": "go", "aliases": "go", "marker": "go.mod"},
				{"runtime": "python", "aliases": "python", "marker": "requirements.txt, pyproject.toml, setup.py"},
				{"runtime": "typescript", "aliases": "typescript, node, bun, js", "marker": "package.json, bun.lockb"},
				{"runtime": "ruby", "aliases": "ruby", "marker": "Gemfile"},
				{"runtime": "rust", "aliases": "rust", "marker": "Cargo.toml"},
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(runtimes)
		},
	}
}

func newSchemaJobCommand(_ *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "job",
		Short: "Print the schema for a Strait job resource",
		RunE: func(_ *cobra.Command, _ []string) error {
			s := schemaResource{
				Resource:    "job",
				Description: "A Strait job definition. Jobs are the primary scheduling unit.",
				Fields: []schemaField{
					{Name: "id", Type: "string", Required: false, Description: "Unique job identifier (server-assigned UUID)"},
					{Name: "name", Type: "string", Required: true, Description: "Human-readable job name"},
					{Name: "slug", Type: "string", Required: true, Description: "URL-safe identifier used in CLI commands and API paths"},
					{Name: "project_id", Type: "string", Required: true, Description: "Project the job belongs to"},
					{Name: "kind", Type: "string", Required: true, Description: "Job manifest kind", Enum: []string{"job"}},
					{Name: "endpoint_url", Type: "string", Required: false, Description: "HTTP endpoint invoked on each run (endpoint-based jobs)"},
					{Name: "cron", Type: "string", Required: false, Description: "Cron expression for scheduled runs (e.g. 0 * * * *)"},
					{Name: "timezone", Type: "string", Required: false, Description: "IANA timezone for cron evaluation (e.g. America/New_York)"},
					{Name: "max_retries", Type: "integer", Required: false, Description: "Maximum retry attempts on failure (default: 0)"},
					{Name: "timeout_secs", Type: "integer", Required: false, Description: "Run timeout in seconds"},
					{Name: "concurrency_limit", Type: "integer", Required: false, Description: "Maximum simultaneous runs"},
					{Name: "paused", Type: "boolean", Required: false, Description: "Whether the job is paused (no new runs scheduled)"},
					{Name: "source_type", Type: "string", Required: false, Description: "Deployment source type", Enum: []string{"endpoint", "code"}},
					{Name: "active_deployment_id", Type: "string", Required: false, Description: "Active code deployment ID (code-first jobs only)"},
					{Name: "created_at", Type: "string (RFC3339)", Required: false, Description: "Creation timestamp"},
					{Name: "updated_at", Type: "string (RFC3339)", Required: false, Description: "Last update timestamp"},
				},
			}
			return printSchema(s)
		},
	}
}

func newSchemaDeploymentCommand(_ *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "deployment",
		Short: "Print the schema for a Strait code deployment resource",
		RunE: func(_ *cobra.Command, _ []string) error {
			s := schemaResource{
				Resource:    "deployment",
				Description: "A code-first deployment created by `strait deploy source`.",
				Fields: []schemaField{
					{Name: "id", Type: "string", Required: false, Description: "Unique deployment identifier (server-assigned UUID)"},
					{Name: "job_id", Type: "string", Required: true, Description: "Job this deployment belongs to"},
					{Name: "project_id", Type: "string", Required: true, Description: "Project the deployment belongs to"},
					{Name: "runtime", Type: "string", Required: true, Description: "Language runtime", Enum: []string{"go", "python", "typescript", "ruby", "rust"}},
					{Name: "source_hash", Type: "string", Required: true, Description: "SHA-256 hex digest of the packed source tarball"},
					{Name: "source_size_bytes", Type: "integer", Required: true, Description: "Size of the packed source tarball in bytes"},
					{Name: "version", Type: "integer", Required: false, Description: "Monotonically increasing deployment version number"},
					{Name: "status", Type: "string", Required: false, Description: "Current deployment status", Enum: []string{"pending", "building", "ready", "failed", "timed_out"}},
					{Name: "built_image_uri", Type: "string", Required: false, Description: "OCI image URI after a successful build"},
					{Name: "error_message", Type: "string", Required: false, Description: "Human-readable build error (failed/timed_out deployments only)"},
					{Name: "created_at", Type: "string (RFC3339)", Required: false, Description: "Creation timestamp"},
					{Name: "finished_at", Type: "string (RFC3339)", Required: false, Description: "Build completion timestamp"},
				},
			}
			return printSchema(s)
		},
	}
}

func newSchemaWorkflowCommand(_ *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "workflow",
		Short: "Print the schema for a Strait workflow resource",
		RunE: func(_ *cobra.Command, _ []string) error {
			s := schemaResource{
				Resource:    "workflow",
				Description: "A Strait workflow — a directed acyclic graph of job steps.",
				Fields: []schemaField{
					{Name: "id", Type: "string", Required: false, Description: "Unique workflow identifier"},
					{Name: "name", Type: "string", Required: true, Description: "Human-readable workflow name"},
					{Name: "slug", Type: "string", Required: true, Description: "URL-safe identifier"},
					{Name: "project_id", Type: "string", Required: true, Description: "Project the workflow belongs to"},
					{Name: "steps", Type: "array", Required: true, Description: "Ordered list of workflow steps; each step references a job by slug"},
					{Name: "cron", Type: "string", Required: false, Description: "Cron expression for scheduled workflow runs"},
					{Name: "timezone", Type: "string", Required: false, Description: "IANA timezone for cron evaluation"},
					{Name: "paused", Type: "boolean", Required: false, Description: "Whether the workflow is paused"},
					{Name: "created_at", Type: "string (RFC3339)", Required: false, Description: "Creation timestamp"},
					{Name: "updated_at", Type: "string (RFC3339)", Required: false, Description: "Last update timestamp"},
				},
			}
			return printSchema(s)
		},
	}
}

func newSchemaRunCommand(_ *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Print the schema for a Strait job run resource",
		RunE: func(_ *cobra.Command, _ []string) error {
			s := schemaResource{
				Resource:    "run",
				Description: "A single execution instance of a Strait job.",
				Fields: []schemaField{
					{Name: "id", Type: "string", Required: false, Description: "Unique run identifier"},
					{Name: "job_id", Type: "string", Required: true, Description: "Parent job ID"},
					{Name: "project_id", Type: "string", Required: true, Description: "Project the run belongs to"},
					{Name: "status", Type: "string", Required: false, Description: "Current run status",
						Enum: []string{"delayed", "queued", "dequeued", "executing", "waiting", "completed", "failed", "timed_out", "crashed", "system_failed", "canceled", "expired", "dead_letter", "replay_staged", "paused"}},
					{Name: "payload", Type: "object", Required: false, Description: "Arbitrary JSON payload passed to the job endpoint"},
					{Name: "result", Type: "object", Required: false, Description: "Arbitrary JSON result returned by the job endpoint"},
					{Name: "error", Type: "string", Required: false, Description: "Error message for failed runs"},
					{Name: "attempt", Type: "integer", Required: false, Description: "Current attempt number (1-based)"},
					{Name: "max_retries", Type: "integer", Required: false, Description: "Maximum retries configured at run time"},
					{Name: "scheduled_at", Type: "string (RFC3339)", Required: false, Description: "When the run was scheduled"},
					{Name: "started_at", Type: "string (RFC3339)", Required: false, Description: "When execution began"},
					{Name: "finished_at", Type: "string (RFC3339)", Required: false, Description: "When execution ended"},
					{Name: "created_at", Type: "string (RFC3339)", Required: false, Description: "Creation timestamp"},
				},
			}
			return printSchema(s)
		},
	}
}

func printSchema(s schemaResource) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		return fmt.Errorf("encode schema: %w", err)
	}
	return nil
}
