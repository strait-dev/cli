package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newJobsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "Manage jobs",
		Long:  idOrSlugLong("job", "Manage jobs."),
	}

	getCmd := newJobsGetCommand(state)
	getCmd.ValidArgsFunction = completeJobSlugs(state)
	deleteCmd := newJobsDeleteCommand(state)
	deleteCmd.ValidArgsFunction = completeJobSlugs(state)
	triggerCmd := newJobsTriggerCommand(state)
	triggerCmd.ValidArgsFunction = completeJobSlugs(state)
	versionsCmd := newJobsVersionsCommand(state)
	versionsCmd.ValidArgsFunction = completeJobSlugs(state)
	describeCmd := newJobsDescribeCommand(state)
	describeCmd.ValidArgsFunction = completeJobSlugs(state)
	editCmd := newJobsEditCommand(state)
	editCmd.ValidArgsFunction = completeJobSlugs(state)
	bulkTriggerCmd := newJobsTriggerBulkCommand(state)
	bulkTriggerCmd.ValidArgsFunction = completeJobSlugs(state)
	pauseCmd := newJobsPauseCommand(state)
	pauseCmd.ValidArgsFunction = completeJobSlugs(state)
	resumeCmd := newJobsResumeCommand(state)
	resumeCmd.ValidArgsFunction = completeJobSlugs(state)
	updateCmd := newJobsUpdateCommand(state)
	updateCmd.ValidArgsFunction = completeJobSlugs(state)
	cloneCmd := newJobsCloneCommand(state)
	cloneCmd.ValidArgsFunction = completeJobSlugs(state)
	healthCmd := newJobsHealthCommand(state)
	healthCmd.ValidArgsFunction = completeJobSlugs(state)
	dependenciesCmd := newJobsDependenciesCommand(state)
	dependenciesCmd.ValidArgsFunction = completeJobSlugs(state)
	addDependencyCmd := newJobsAddDependencyCommand(state)
	addDependencyCmd.ValidArgsFunction = completeJobSlugs(state)

	cmd.AddCommand(newJobsListCommand(state))
	cmd.AddCommand(getCmd)
	cmd.AddCommand(newJobsCreateCommand(state))
	cmd.AddCommand(triggerCmd)
	cmd.AddCommand(bulkTriggerCmd)
	cmd.AddCommand(deleteCmd)
	cmd.AddCommand(versionsCmd)
	cmd.AddCommand(describeCmd)
	cmd.AddCommand(editCmd)
	cmd.AddCommand(pauseCmd)
	cmd.AddCommand(resumeCmd)
	cmd.AddCommand(updateCmd)
	cmd.AddCommand(cloneCmd)
	cmd.AddCommand(healthCmd)
	cmd.AddCommand(dependenciesCmd)
	cmd.AddCommand(addDependencyCmd)
	cmd.AddCommand(newJobsBatchCommand(state))

	return cmd
}

func newJobsDeleteCommand(state *appState) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <job-id-or-slug>",
		Short: "Delete a job by ID or slug",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfirmation(state, "Delete this job?", yes); err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			jobID, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			if err := cli.DeleteJob(cmd.Context(), jobID); err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Deleted job "+styles.Bold.Render(jobID)))
				return nil
			}
			return printData(state, map[string]any{"deleted": true, "id": jobID})
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")

	return cmd
}

func newJobsVersionsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versions <job-id-or-slug>",
		Short: "List version history for a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			jobID, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			versions, err := cli.ListJobVersions(cmd.Context(), jobID)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				for _, v := range versions {
					fmt.Fprintf(os.Stderr, "  %s  v%-4d  %s\n",
						styles.MutedStyle.Render(v.ID),
						v.Version,
						styles.MutedStyle.Render(v.CreatedAt.Format("2006-01-02 15:04:05")),
					)
				}
				return nil
			}
			return printData(state, versions)
		},
	}

	return cmd
}

func newJobsDescribeCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <job-id-or-slug>",
		Short: "Show rich details and recent runs for a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			jobID, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			job, err := cli.GetJob(cmd.Context(), jobID)
			if err != nil {
				return err
			}

			runs, err := cli.ListRuns(cmd.Context(), job.ProjectID, "", 100, nil)
			if err != nil {
				return err
			}

			recent := make([]map[string]any, 0, 10)
			for _, run := range runs {
				if run.JobID != job.ID {
					continue
				}
				recent = append(recent, map[string]any{
					"id":          run.ID,
					"status":      run.Status,
					"attempt":     run.Attempt,
					"triggeredBy": run.TriggeredBy,
					"createdAt":   run.CreatedAt,
				})
				if len(recent) >= 10 {
					break
				}
			}

			payload := map[string]any{
				"job":         job,
				"recent_runs": recent,
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("ID", job.ID),
					styles.DetailLine("Name", job.Name),
					styles.DetailLine("Slug", job.Slug),
					styles.DetailLine("Enabled", styles.Enabled(job.Enabled)),
					styles.DetailLine("Source", jobSourceDisplay(job.SourceType)),
					styles.DetailLine("Endpoint", job.EndpointURL),
					styles.DetailLine("Active Deploy", job.ActiveDeploymentID),
					styles.DetailLine("Cron", job.Cron),
					styles.DetailLine("Timeout", fmt.Sprintf("%ds", job.TimeoutSecs)),
					styles.DetailLine("Max Retry", fmt.Sprintf("%d", job.MaxAttempts)),
					styles.DetailLine("Version", fmt.Sprintf("%d", job.Version)),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Job Details", lines))
				if len(recent) > 0 {
					fmt.Fprintln(os.Stderr)
					fmt.Fprintln(os.Stderr, styles.SectionHeader("Recent Runs", len(recent)))
					for _, r := range recent {
						status, _ := r["status"].(string)
						id, _ := r["id"].(string)
						attempt, _ := r["attempt"].(int)
						triggeredBy, _ := r["triggeredBy"].(string)
						fmt.Fprintf(os.Stderr, "  %s  %s  attempt=%d  by=%s\n",
							styles.StatusBadge(status), id, attempt, triggeredBy)
					}
				}
				return nil
			}
			return printData(state, payload)
		},
	}

	return cmd
}

func newJobsEditCommand(state *appState) *cobra.Command {
	var field string
	var editor string

	cmd := &cobra.Command{
		Use:   "edit <job-id-or-slug>",
		Short: "Edit a job via --field or interactive editor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			jobID, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			if strings.TrimSpace(field) == "" {
				return runInteractiveJobEdit(cmd.Context(), cli, state, jobID, editor)
			}

			parts := strings.SplitN(field, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid --field format, expected key=value")
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			upd := client.UpdateJobRequest{}
			switch key {
			case "name":
				upd.Name = &val
			case "slug":
				upd.Slug = &val
			case "description":
				upd.Description = &val
			case "cron":
				upd.Cron = &val
			case "endpoint_url", "endpoint":
				upd.EndpointURL = &val
			case "enabled":
				parsed, err := strconv.ParseBool(val)
				if err != nil {
					return fmt.Errorf("enabled must be true|false")
				}
				upd.Enabled = &parsed
			case "max_attempts":
				parsed, err := strconv.Atoi(val)
				if err != nil {
					return fmt.Errorf("max_attempts must be an integer")
				}
				upd.MaxAttempts = &parsed
			case "timeout_secs":
				parsed, err := strconv.Atoi(val)
				if err != nil {
					return fmt.Errorf("timeout_secs must be an integer")
				}
				upd.TimeoutSecs = &parsed
			case "run_ttl_secs":
				parsed, err := strconv.Atoi(val)
				if err != nil {
					return fmt.Errorf("run_ttl_secs must be an integer")
				}
				upd.RunTTLSecs = &parsed
			default:
				return fmt.Errorf("unsupported field %q", key)
			}

			job, err := cli.UpdateJob(cmd.Context(), jobID, upd)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Updated job "+styles.Bold.Render(job.ID)+" (version "+fmt.Sprintf("%d", job.Version)+")"))
				return nil
			}
			return printData(state, job)
		},
	}

	cmd.Flags().StringVar(&field, "field", "", "field update in key=value form")
	cmd.Flags().StringVar(&editor, "editor", "", "editor command for interactive mode")

	return cmd
}

type editableJob struct {
	Name        string `yaml:"name"`
	Slug        string `yaml:"slug"`
	Description string `yaml:"description,omitempty"`
	Cron        string `yaml:"cron,omitempty"`
	EndpointURL string `yaml:"endpoint_url"`
	MaxAttempts int    `yaml:"max_attempts"`
	TimeoutSecs int    `yaml:"timeout_secs"`
	RunTTLSecs  int    `yaml:"run_ttl_secs,omitempty"`
	Enabled     bool   `yaml:"enabled"`
}

func runInteractiveJobEdit(ctx context.Context, cli *client.Client, state *appState, jobID, editorOverride string) error {
	job, err := cli.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	original := editableJob{
		Name:        job.Name,
		Slug:        job.Slug,
		Description: job.Description,
		Cron:        job.Cron,
		EndpointURL: job.EndpointURL,
		MaxAttempts: job.MaxAttempts,
		TimeoutSecs: job.TimeoutSecs,
		RunTTLSecs:  job.RunTTLSecs,
		Enabled:     job.Enabled,
	}

	tmp, err := os.CreateTemp("", "strait-job-edit-*.yaml")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if closeErr := tmp.Close(); closeErr != nil {
		return closeErr
	}
	defer os.Remove(tmpPath)

	encoded, err := yaml.Marshal(original)
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmpPath, encoded, 0o600); err != nil {
		return err
	}

	editor := strings.TrimSpace(editorOverride)
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, tmpPath) //nolint:gosec // editor is from $EDITOR env var or default vi, tmpPath is a temp file
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout // printdata-ok: subprocess stdout passthrough for $EDITOR
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	raw, err := os.ReadFile(tmpPath) //nolint:gosec // tmpPath is a temp file created by this function
	if err != nil {
		return err
	}
	updated := editableJob{}
	if err := yaml.Unmarshal(raw, &updated); err != nil {
		return err
	}

	if updated == original {
		if isTTYRich(state) {
			fmt.Fprintln(os.Stderr, styles.Info("No changes detected"))
			return nil
		}
		return printData(state, map[string]any{"updated": false, "reason": "no changes"})
	}

	upd := client.UpdateJobRequest{
		Name:        &updated.Name,
		Slug:        &updated.Slug,
		Description: &updated.Description,
		Cron:        &updated.Cron,
		EndpointURL: &updated.EndpointURL,
		MaxAttempts: &updated.MaxAttempts,
		TimeoutSecs: &updated.TimeoutSecs,
		RunTTLSecs:  &updated.RunTTLSecs,
		Enabled:     &updated.Enabled,
	}
	patched, err := cli.UpdateJob(ctx, jobID, upd)
	if err != nil {
		return err
	}

	if isTTYRich(state) {
		fmt.Fprintln(os.Stderr, styles.Success("Updated job "+styles.Bold.Render(patched.ID)+" (version "+fmt.Sprintf("%d", patched.Version)+")"))
		return nil
	}
	return printData(state, patched)
}

// jobSourceDisplay returns a short label for a job's source type, e.g. "code" or "endpoint".
func jobSourceDisplay(sourceType string) string {
	switch sourceType {
	case "code":
		return "code"
	case "endpoint", "":
		return "endpoint"
	default:
		return sourceType
	}
}

func newJobsListCommand(state *appState) *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List jobs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			projectID, err = requireProjectID(state, projectID)
			if err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			jobs, err := cli.ListJobs(cmd.Context(), projectID)
			if err != nil {
				return err
			}

			rows := make([]map[string]any, 0, len(jobs))
			for _, job := range jobs {
				row := map[string]any{
					"id":          job.ID,
					"name":        job.Name,
					"slug":        job.Slug,
					"cron":        job.Cron,
					"enabled":     job.Enabled,
					"source_type": jobSourceDisplay(job.SourceType),
				}
				if job.ActiveDeploymentID != "" {
					row["active_deployment_id"] = job.ActiveDeploymentID
				}
				rows = append(rows, row)
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("Jobs", len(jobs)))
				for _, job := range jobs {
					cron := job.Cron
					if cron == "" {
						cron = "--"
					}
					src := jobSourceDisplay(job.SourceType)
					fmt.Fprintf(os.Stderr, "  %s  %-20s  source=%-8s  cron=%s  %s\n",
						styles.Enabled(job.Enabled),
						styles.Bold.Render(job.Slug),
						styles.MutedStyle.Render(src),
						styles.MutedStyle.Render(cron),
						styles.MutedStyle.Render(job.ID),
					)
				}
				return nil
			}
			return printData(state, rows)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")

	return cmd
}

func newJobsGetCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <job-id-or-slug>",
		Short: "Get a job by ID or slug",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}
			jobID, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			job, err := cli.GetJob(cmd.Context(), jobID)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("ID", job.ID),
					styles.DetailLine("Name", job.Name),
					styles.DetailLine("Slug", job.Slug),
					styles.DetailLine("Enabled", styles.Enabled(job.Enabled)),
					styles.DetailLine("Source", jobSourceDisplay(job.SourceType)),
					styles.DetailLine("Endpoint", job.EndpointURL),
					styles.DetailLine("Active Deploy", job.ActiveDeploymentID),
					styles.DetailLine("Cron", job.Cron),
					styles.DetailLine("Timeout", fmt.Sprintf("%ds", job.TimeoutSecs)),
					styles.DetailLine("Max Retry", fmt.Sprintf("%d", job.MaxAttempts)),
					styles.DetailLine("Version", fmt.Sprintf("%d", job.Version)),
				}
				fmt.Fprint(os.Stderr, styles.DetailBox("Job", lines))
				return nil
			}
			return printData(state, job)
		},
	}

	return cmd
}

func newJobsCreateCommand(state *appState) *cobra.Command {
	var req client.CreateJobRequest
	var idempotencyKey string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a job",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if req.ProjectID == "" {
				req.ProjectID = state.opts.projectID
			}
			if req.ProjectID == "" || req.Name == "" || req.Slug == "" {
				return fmt.Errorf("project, name, and slug are required")
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			job, err := cli.CreateJob(cmd.Context(), req, idempotencyKey)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Created job "+styles.Bold.Render(job.Slug)))
				fmt.Fprintln(os.Stderr, styles.KeyValue("ID", job.ID))
				return nil
			}
			return printData(state, job)
		},
	}

	cmd.Flags().StringVar(&req.ProjectID, "project", "", "project ID")
	cmd.Flags().StringVar(&req.Name, "name", "", "job name")
	cmd.Flags().StringVar(&req.Slug, "slug", "", "job slug")
	cmd.Flags().StringVar(&req.Description, "description", "", "job description")
	cmd.Flags().StringVar(&req.Cron, "cron", "", "cron schedule")
	cmd.Flags().StringVar(&req.EndpointURL, "endpoint", "", "job endpoint URL")
	cmd.Flags().IntVar(&req.TimeoutSecs, "timeout-secs", 60, "execution timeout in seconds")
	cmd.Flags().IntVar(&req.MaxAttempts, "max-attempts", 3, "max attempts")
	cmd.Flags().IntVar(&req.RunTTLSecs, "run-ttl-secs", 0, "run TTL in seconds")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "idempotency key to prevent duplicate creates (passed as X-Idempotency-Key header)")

	return cmd
}

func newJobsTriggerBulkCommand(state *appState) *cobra.Command {
	var itemsJSON string
	var itemsFile string

	cmd := &cobra.Command{
		Use:   "trigger-bulk <job-id-or-slug>",
		Short: "Trigger multiple runs for a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(itemsJSON) == "" && strings.TrimSpace(itemsFile) == "" {
				return fmt.Errorf("either --items-json or --items-file is required")
			}
			if strings.TrimSpace(itemsJSON) != "" && strings.TrimSpace(itemsFile) != "" {
				return fmt.Errorf("use either --items-json or --items-file, not both")
			}

			raw := strings.TrimSpace(itemsJSON)
			if strings.TrimSpace(itemsFile) != "" {
				fileContent, err := os.ReadFile(itemsFile) //nolint:gosec // itemsFile is from --items-file CLI flag
				if err != nil {
					return err
				}
				raw = strings.TrimSpace(string(fileContent))
			}

			items := make([]client.BulkTriggerItem, 0)
			if err := json.Unmarshal([]byte(raw), &items); err != nil {
				return fmt.Errorf("invalid items payload: %w", err)
			}
			if len(items) == 0 {
				return fmt.Errorf("items array must not be empty")
			}
			for _, item := range items {
				if len(item.Payload) > 0 && !json.Valid(item.Payload) {
					return fmt.Errorf("all item payloads must be valid JSON")
				}
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			jobID, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			resp, err := cli.BulkTriggerJob(cmd.Context(), jobID, client.BulkTriggerRequest{Items: items})
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success(fmt.Sprintf("Triggered %d runs (%d total)", resp.Created, resp.Total)))
				return nil
			}
			return printData(state, resp)
		},
	}

	cmd.Flags().StringVar(&itemsJSON, "items-json", "", "JSON array of bulk trigger items")
	cmd.Flags().StringVar(&itemsFile, "items-file", "", "path to JSON file containing bulk trigger items array")

	return cmd
}

func newJobsTriggerCommand(state *appState) *cobra.Command {
	var payload string
	var payloadFile string
	var priority int
	var scheduledAt string
	var idempotencyKey string

	cmd := &cobra.Command{
		Use:   "trigger <job-id-or-slug>",
		Short: "Trigger a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := client.TriggerJobRequest{Priority: priority}

			if strings.TrimSpace(scheduledAt) != "" {
				ts, err := time.Parse(time.RFC3339, scheduledAt)
				if err != nil {
					return fmt.Errorf("invalid scheduled-at: %w", err)
				}
				req.ScheduledAt = &ts
			}

			if payloadFile != "" {
				raw, err := os.ReadFile(payloadFile) //nolint:gosec // user-provided local file path for explicit CLI input
				if err != nil {
					return err
				}
				req.Payload = json.RawMessage(raw)
			} else if strings.TrimSpace(payload) != "" {
				req.Payload = json.RawMessage(payload)
			}

			if len(req.Payload) > 0 && !json.Valid(req.Payload) {
				return fmt.Errorf("payload must be valid JSON")
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			jobID, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			resp, err := cli.TriggerJob(cmd.Context(), jobID, req, idempotencyKey)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Info("Triggered run "+styles.Bold.Render(resp.ID)))
				return nil
			}
			return printData(state, resp)
		},
	}

	cmd.Flags().StringVar(&payload, "payload", "", "inline JSON payload")
	cmd.Flags().StringVar(&payloadFile, "payload-file", "", "path to payload JSON file")
	cmd.Flags().IntVar(&priority, "priority", 0, "run priority")
	cmd.Flags().StringVar(&scheduledAt, "scheduled-at", "", "RFC3339 timestamp")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "idempotency key")

	return cmd
}

func newJobsPauseCommand(state *appState) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "pause <job-id-or-slug>",
		Short: "Pause a job (no new runs will be scheduled)",
		Args:  cobra.ExactArgs(1),
		Example: `  strait jobs pause my-job
  strait jobs pause my-job --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfirmation(state, "Pause this job?", yes); err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			jobID, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			enabled := false
			job, err := cli.UpdateJob(cmd.Context(), jobID, client.UpdateJobRequest{Enabled: &enabled})
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Paused job "+styles.Bold.Render(job.ID)))
				return nil
			}
			return printData(state, job)
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")
	return cmd
}

func newJobsResumeCommand(state *appState) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "resume <job-id-or-slug>",
		Short: "Resume a paused job",
		Args:  cobra.ExactArgs(1),
		Example: `  strait jobs resume my-job
  strait jobs resume my-job --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfirmation(state, "Resume this job?", yes); err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			jobID, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			enabled := true
			job, err := cli.UpdateJob(cmd.Context(), jobID, client.UpdateJobRequest{Enabled: &enabled})
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Resumed job "+styles.Bold.Render(job.ID)))
				return nil
			}
			return printData(state, job)
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")
	return cmd
}

func newJobsUpdateCommand(state *appState) *cobra.Command {
	var fields []string
	var fromStdin bool

	cmd := &cobra.Command{
		Use:   "update <job-id-or-slug>",
		Short: "Update job fields non-interactively",
		Long: `Apply field updates to a job without opening an editor.
Accepts --field key=value flags (repeatable) or --stdin to read a JSON patch from stdin.
Designed for scripting and CI; never prompts for input.`,
		Args: cobra.ExactArgs(1),
		Example: `  strait jobs update my-job --field name=new-name
  strait jobs update my-job --field cron="0 * * * *" --field timeout_secs=120
  echo '{"endpoint_url":"http://localhost:3000/jobs/my-job"}' | strait jobs update my-job --stdin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate flags before making any API calls.
			upd := client.UpdateJobRequest{}
			for _, f := range fields {
				parts := strings.SplitN(f, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --field %q: expected key=value", f)
				}
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if err := applyJobField(&upd, key, val); err != nil {
					return err
				}
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			jobID, err := resolveJobIdentifier(cmd.Context(), cli, state, args[0])
			if err != nil {
				return err
			}

			if fromStdin {
				var patch map[string]any
				if err := json.NewDecoder(os.Stdin).Decode(&patch); err != nil {
					return fmt.Errorf("read stdin: %w", err)
				}
				if err := applyJobPatch(&upd, patch); err != nil {
					return err
				}
			}

			job, err := cli.UpdateJob(cmd.Context(), jobID, upd)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Updated job "+styles.Bold.Render(job.ID)+" (version "+fmt.Sprintf("%d", job.Version)+")"))
				return nil
			}
			return printData(state, job)
		},
	}

	cmd.Flags().StringArrayVar(&fields, "field", nil, "field update in key=value form (repeatable)")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "read JSON patch from stdin")
	return cmd
}

// applyJobField applies a single key=value string to an UpdateJobRequest.
func applyJobField(upd *client.UpdateJobRequest, key, val string) error {
	switch key {
	case "name":
		upd.Name = &val
	case "slug":
		upd.Slug = &val
	case "description":
		upd.Description = &val
	case "cron":
		upd.Cron = &val
	case "endpoint_url", "endpoint":
		upd.EndpointURL = &val
	case "enabled":
		parsed, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("enabled must be true|false")
		}
		upd.Enabled = &parsed
	case "max_attempts":
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("max_attempts must be an integer")
		}
		upd.MaxAttempts = &parsed
	case "timeout_secs":
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("timeout_secs must be an integer")
		}
		upd.TimeoutSecs = &parsed
	case "run_ttl_secs":
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("run_ttl_secs must be an integer")
		}
		upd.RunTTLSecs = &parsed
	default:
		return fmt.Errorf("unsupported field %q (supported: name, slug, description, cron, endpoint_url, enabled, max_attempts, timeout_secs, run_ttl_secs)", key)
	}
	return nil
}

// applyJobPatch applies a JSON map of field updates to an UpdateJobRequest.
func applyJobPatch(upd *client.UpdateJobRequest, patch map[string]any) error {
	for k, v := range patch {
		val := fmt.Sprintf("%v", v)
		if err := applyJobField(upd, k, val); err != nil {
			return err
		}
	}
	return nil
}
