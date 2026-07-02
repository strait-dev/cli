package main

import (
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
		Example: `  strait schema job
  strait schema deployment
  strait schema workflow
  strait schema run
  strait schema trigger
  strait schema secret
  strait schema api-key`,
	}

	cmd.AddCommand(newSchemaJobCommand(state))
	cmd.AddCommand(newSchemaDeploymentCommand(state))
	cmd.AddCommand(newSchemaWorkflowCommand(state))
	cmd.AddCommand(newSchemaRunCommand(state))
	cmd.AddCommand(newSchemaTriggerCommand(state))
	cmd.AddCommand(newSchemaSecretCommand(state))
	cmd.AddCommand(newSchemaAPIKeyCommand(state))

	return cmd
}

func newSchemaJobCommand(state *appState) *cobra.Command {
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
					{Name: "active_deployment_id", Type: "string", Required: false, Description: "Active deployment version ID"},
					{Name: "created_at", Type: "string (RFC3339)", Required: false, Description: "Creation timestamp"},
					{Name: "updated_at", Type: "string (RFC3339)", Required: false, Description: "Last update timestamp"},
				},
			}
			return printData(state, s)
		},
	}
}

func newSchemaDeploymentCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "deployment",
		Short: "Print the schema for a Strait deployment version resource",
		RunE: func(_ *cobra.Command, _ []string) error {
			s := schemaResource{
				Resource:    "deployment",
				Description: "A manifest deployment version created by `strait deploy`.",
				Fields: []schemaField{
					{Name: "id", Type: "string", Required: false, Description: "Unique deployment identifier (server-assigned UUID)"},
					{Name: "project_id", Type: "string", Required: true, Description: "Project the deployment belongs to"},
					{Name: "environment", Type: "string", Required: true, Description: "Deployment environment"},
					{Name: "runtime", Type: "string", Required: true, Description: "Runtime declared by the manifest"},
					{Name: "artifact_uri", Type: "string", Required: true, Description: "Pre-built artifact URI"},
					{Name: "strategy", Type: "string", Required: false, Description: "Rollout strategy", Enum: []string{"direct", "canary"}},
					{Name: "canary_percent", Type: "integer", Required: false, Description: "Canary traffic percentage"},
					{Name: "canary_duration", Type: "string", Required: false, Description: "Canary duration"},
					{Name: "status", Type: "string", Required: false, Description: "Current deployment status"},
					{Name: "checksum", Type: "string", Required: false, Description: "Manifest checksum"},
					{Name: "created_at", Type: "string (RFC3339)", Required: false, Description: "Creation timestamp"},
				},
			}
			return printData(state, s)
		},
	}
}

func newSchemaWorkflowCommand(state *appState) *cobra.Command {
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
			return printData(state, s)
		},
	}
}

func newSchemaRunCommand(state *appState) *cobra.Command {
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
			return printData(state, s)
		},
	}
}

func newSchemaTriggerCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "trigger",
		Short: "Print the schema for a Strait trigger resource",
		RunE: func(_ *cobra.Command, _ []string) error {
			s := schemaResource{
				Resource:    "trigger",
				Description: "An event trigger that fires a job or workflow run when a matching event is received.",
				Fields: []schemaField{
					{Name: "id", Type: "string", Required: false, Description: "Unique trigger identifier (server-assigned UUID)"},
					{Name: "name", Type: "string", Required: true, Description: "Human-readable trigger name"},
					{Name: "slug", Type: "string", Required: true, Description: "URL-safe identifier used in API paths"},
					{Name: "project_id", Type: "string", Required: true, Description: "Project the trigger belongs to"},
					{Name: "event", Type: "string", Required: true, Description: "Event name this trigger listens for"},
					{Name: "filter", Type: "object", Required: false, Description: "CEL or JSON-match filter applied to incoming event payload"},
					{Name: "target_type", Type: "string", Required: true, Description: "Type of target to fire on match", Enum: []string{"job", "workflow"}},
					{Name: "target_id", Type: "string", Required: true, Description: "ID of the job or workflow to trigger"},
					{Name: "payload_template", Type: "object", Required: false, Description: "Go template applied to event payload to produce the run payload"},
					{Name: "enabled", Type: "boolean", Required: false, Description: "Whether the trigger is active (default: true)"},
					{Name: "created_at", Type: "string (RFC3339)", Required: false, Description: "Creation timestamp"},
					{Name: "updated_at", Type: "string (RFC3339)", Required: false, Description: "Last update timestamp"},
				},
			}
			return printData(state, s)
		},
	}
}

func newSchemaSecretCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "secret",
		Short: "Print the schema for a Strait secret resource",
		RunE: func(_ *cobra.Command, _ []string) error {
			s := schemaResource{
				Resource:    "secret",
				Description: "A named secret value scoped to a project, injected as an environment variable at run time.",
				Fields: []schemaField{
					{Name: "id", Type: "string", Required: false, Description: "Unique secret identifier (server-assigned UUID)"},
					{Name: "key", Type: "string", Required: true, Description: "Environment variable name (e.g. DATABASE_URL); must be uppercase with underscores"},
					{Name: "value", Type: "string", Required: true, Description: "Secret value; write-only — never returned by the API after creation"},
					{Name: "project_id", Type: "string", Required: true, Description: "Project the secret belongs to"},
					{Name: "description", Type: "string", Required: false, Description: "Optional human-readable description"},
					{Name: "created_at", Type: "string (RFC3339)", Required: false, Description: "Creation timestamp"},
					{Name: "updated_at", Type: "string (RFC3339)", Required: false, Description: "Last update timestamp"},
				},
			}
			return printData(state, s)
		},
	}
}

func newSchemaAPIKeyCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "api-key",
		Short: "Print the schema for a Strait API key resource",
		RunE: func(_ *cobra.Command, _ []string) error {
			s := schemaResource{
				Resource:    "api_key",
				Description: "An API key used to authenticate CLI and programmatic access to the Strait API.",
				Fields: []schemaField{
					{Name: "id", Type: "string", Required: false, Description: "Unique API key identifier"},
					{Name: "name", Type: "string", Required: true, Description: "Human-readable label for the key"},
					{Name: "prefix", Type: "string", Required: false, Description: "First 8 characters of the key token (for identification; full token only returned on creation)"},
					{Name: "scopes", Type: "array", Required: false, Description: "Permission scopes granted to this key", Enum: []string{"read", "write", "admin"}},
					{Name: "project_id", Type: "string", Required: false, Description: "Project scope (omit for org-level key)"},
					{Name: "expires_at", Type: "string (RFC3339)", Required: false, Description: "Expiry timestamp (null = never expires)"},
					{Name: "last_used_at", Type: "string (RFC3339)", Required: false, Description: "When the key was last used to authenticate"},
					{Name: "created_at", Type: "string (RFC3339)", Required: false, Description: "Creation timestamp"},
				},
			}
			return printData(state, s)
		},
	}
}
