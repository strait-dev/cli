package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/types"

	"github.com/spf13/cobra"
)

const (
	defaultProjectConfigName = "strait.json"
	legacyDeployConfigName   = "strait.deploy.json"
)

// ProjectConfig is the canonical on-disk orchestration file consumed by
// `strait sync` and `strait dev`. It describes resources Strait orchestrates;
// it does not describe how customer code is built, hosted, or deployed.
type ProjectConfig struct {
	SchemaURL string            `json:"$schema,omitempty"`
	Version   string            `json:"version"`
	Jobs      []ProjectJob      `json:"jobs"`
	Workflows []ProjectWorkflow `json:"workflows,omitempty"`
	Metadata  map[string]any    `json:"metadata,omitempty"`
}

type ProjectJob struct {
	Slug        string          `json:"slug"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	EndpointURL string          `json:"endpoint_url"`
	Cron        string          `json:"cron,omitempty"`
	MaxAttempts int             `json:"max_attempts,omitempty"`
	TimeoutSecs int             `json:"timeout_secs,omitempty"`
	RunTTLSecs  int             `json:"run_ttl_secs,omitempty"`
	Schema      json.RawMessage `json:"payload_schema,omitempty"`
}

type ProjectWorkflow struct {
	Slug        string                `json:"slug"`
	Name        string                `json:"name,omitempty"`
	Description string                `json:"description,omitempty"`
	Enabled     *bool                 `json:"enabled,omitempty"`
	Steps       []ProjectWorkflowStep `json:"steps,omitempty"`
}

type ProjectWorkflowStep struct {
	JobID     string          `json:"job_id,omitempty"`
	JobRef    string          `json:"job_ref,omitempty"`
	StepRef   string          `json:"step_ref"`
	DependsOn []string        `json:"depends_on,omitempty"`
	Condition json.RawMessage `json:"condition,omitempty"`
	OnFailure string          `json:"on_failure,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type ProjectConfigLoadResult struct {
	Config *ProjectConfig
	Path   string
	Legacy bool
}

type SyncResult struct {
	Kind   string `json:"kind"`
	Slug   string `json:"slug"`
	Action string `json:"action"`
	ID     string `json:"id,omitempty"`
	Error  string `json:"error,omitempty"`
}

type SyncSummary struct {
	Created int          `json:"created"`
	Updated int          `json:"updated"`
	Skipped int          `json:"skipped"`
	Failed  int          `json:"failed"`
	Results []SyncResult `json:"results"`
}

type syncOptions struct {
	dir        string
	file       string
	dryRun     bool
	prune      bool
	yes        bool
	legacyMode bool
}

func newSyncCommand(state *appState) *cobra.Command {
	var opts syncOptions
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync strait.json orchestration definitions",
		Long: `Read strait.json from the project root and upsert the jobs and
workflows that Strait orchestrates. Customer code still runs on customer
infrastructure; this command only syncs orchestration metadata.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSyncCommand(cmd, state, opts)
		},
	}
	addSyncFlags(cmd, &opts, false)
	return cmd
}

func newDeployCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deprecated compatibility commands",
		Long:  "Deprecated compatibility surface. Strait is orchestration-only; use `strait sync` to sync strait.json definitions.",
	}
	cmd.AddCommand(newDeployPushCommand(state))
	return cmd
}

func newDeployPushCommand(state *appState) *cobra.Command {
	var opts syncOptions
	opts.legacyMode = true
	cmd := &cobra.Command{
		Use:        "push",
		Short:      "Deprecated: use `strait sync`",
		Long:       "Deprecated compatibility alias for `strait sync`. Reads strait.json by default and syncs orchestration definitions only.",
		Deprecated: "use `strait sync` instead",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSyncCommand(cmd, state, opts)
		},
	}
	addSyncFlags(cmd, &opts, true)
	return cmd
}

func addSyncFlags(cmd *cobra.Command, opts *syncOptions, legacy bool) {
	cmd.Flags().StringVar(&opts.dir, "dir", "", "project root containing strait.json (default: current dir)")
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "explicit path to a strait.json file (overrides --dir)")
	if legacy {
		cmd.Flags().StringVar(&opts.file, "manifest", "", "deprecated alias for --file")
	}
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "print the sync plan without making changes")
	cmd.Flags().BoolVar(&opts.prune, "prune", false, "delete jobs and workflows not present in strait.json (destructive)")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "skip the confirmation prompt for --prune")
}

func runSyncCommand(cmd *cobra.Command, state *appState, opts syncOptions) error {
	projectID := state.opts.projectID
	if strings.TrimSpace(projectID) == "" {
		return fmt.Errorf("project is required (set --project, STRAIT_PROJECT, or context)")
	}

	loaded, err := loadProjectConfig(resolveProjectConfigPath(opts.dir, opts.file))
	if err != nil {
		return err
	}
	if loaded.Legacy && isTTYRich(state) {
		fmt.Fprintln(os.Stderr, styles.Warn("strait.deploy.json is deprecated; rename it to strait.json"))
	}
	if opts.legacyMode && isTTYRich(state) {
		fmt.Fprintln(os.Stderr, styles.Warn("`strait deploy push` is deprecated; use `strait sync`"))
	}

	cli, err := newAPIClient(state)
	if err != nil {
		return err
	}

	summary, err := planSync(cmd.Context(), cli, projectID, loaded.Config)
	if err != nil {
		return err
	}

	if opts.dryRun {
		if isTTYRich(state) {
			renderSyncPlan(summary)
			return nil
		}
		return printData(state, summary)
	}

	if opts.prune {
		if err := requireConfirmation(state, "Pruning removes jobs and workflows not in strait.json. Continue?", opts.yes); err != nil {
			return err
		}
	}

	final, err := applySync(cmd.Context(), cli, projectID, loaded.Config, summary, opts.prune)
	if err != nil {
		return err
	}

	if isTTYRich(state) {
		renderSyncPlan(final)
		if final.Failed > 0 {
			return fmt.Errorf("%d resource(s) failed to sync", final.Failed)
		}
		return nil
	}
	if err := printData(state, final); err != nil {
		return err
	}
	if final.Failed > 0 {
		return fmt.Errorf("%d resource(s) failed to sync", final.Failed)
	}
	return nil
}

func resolveProjectConfigPath(dir, file string) string {
	if strings.TrimSpace(file) != "" {
		return file
	}
	root := dir
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	return filepath.Join(root, defaultProjectConfigName)
}

func loadProjectConfig(path string) (*ProjectConfigLoadResult, error) {
	abs := path
	if !filepath.IsAbs(abs) {
		cwd, _ := os.Getwd()
		abs = filepath.Join(cwd, abs)
	}
	data, err := os.ReadFile(abs) //nolint:gosec // path is user-provided via flag or current project root
	legacy := false
	if err != nil && os.IsNotExist(err) && filepath.Base(path) == defaultProjectConfigName {
		legacyPath := filepath.Join(filepath.Dir(abs), legacyDeployConfigName)
		data, err = os.ReadFile(legacyPath) //nolint:gosec // compatibility path in the same project root
		if err == nil {
			abs = legacyPath
			path = legacyPath
			legacy = true
		}
	}
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project config not found at %s; create strait.json or pass --file <path>", path)
		}
		return nil, fmt.Errorf("read project config: %w", err)
	}
	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid project config JSON: %w", err)
	}
	if len(cfg.Jobs) == 0 && len(cfg.Workflows) == 0 {
		return nil, fmt.Errorf("project config %s has no jobs or workflows", path)
	}
	for i, j := range cfg.Jobs {
		if strings.TrimSpace(j.Slug) == "" {
			return nil, fmt.Errorf("project config jobs[%d]: slug is required", i)
		}
		if strings.TrimSpace(j.EndpointURL) == "" {
			return nil, fmt.Errorf("project config jobs[%d] (%s): endpoint_url is required", i, j.Slug)
		}
	}
	for i, w := range cfg.Workflows {
		if strings.TrimSpace(w.Slug) == "" {
			return nil, fmt.Errorf("project config workflows[%d]: slug is required", i)
		}
	}
	return &ProjectConfigLoadResult{Config: &cfg, Path: abs, Legacy: legacy}, nil
}

func planSync(ctx context.Context, cli *client.Client, projectID string, cfg *ProjectConfig) (*SyncSummary, error) {
	existing, err := cli.ListJobs(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list existing jobs: %w", err)
	}
	bySlug := make(map[string]types.Job, len(existing))
	for _, j := range existing {
		bySlug[j.Slug] = j
	}

	summary := &SyncSummary{}
	for _, job := range cfg.Jobs {
		res := SyncResult{Kind: "job", Slug: job.Slug}
		existingJob, ok := bySlug[job.Slug]
		switch {
		case !ok:
			res.Action = "create"
			summary.Created++
		case !projectJobNeedsUpdate(job, existingJob):
			res.Action = "skip"
			res.ID = existingJob.ID
			summary.Skipped++
		default:
			res.Action = "update"
			res.ID = existingJob.ID
			summary.Updated++
		}
		summary.Results = append(summary.Results, res)
	}

	existingWorkflows, err := cli.ListWorkflows(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list existing workflows: %w", err)
	}
	workflowBySlug := make(map[string]types.Workflow, len(existingWorkflows))
	for _, wf := range existingWorkflows {
		workflowBySlug[wf.Slug] = wf
	}
	for _, wf := range cfg.Workflows {
		res := SyncResult{Kind: "workflow", Slug: wf.Slug}
		existingWorkflow, ok := workflowBySlug[wf.Slug]
		if !ok {
			res.Action = "create"
			summary.Created++
		} else {
			res.Action = "update"
			res.ID = existingWorkflow.ID
			summary.Updated++
		}
		summary.Results = append(summary.Results, res)
	}

	sort.SliceStable(summary.Results, func(i, j int) bool {
		if summary.Results[i].Kind != summary.Results[j].Kind {
			return summary.Results[i].Kind < summary.Results[j].Kind
		}
		return summary.Results[i].Slug < summary.Results[j].Slug
	})
	return summary, nil
}

func applySync(ctx context.Context, cli *client.Client, projectID string, cfg *ProjectConfig, plan *SyncSummary, prune bool) (*SyncSummary, error) {
	out := &SyncSummary{}
	jobBySlug := map[string]ProjectJob{}
	for _, j := range cfg.Jobs {
		jobBySlug[j.Slug] = j
	}
	workflowBySlug := map[string]ProjectWorkflow{}
	for _, w := range cfg.Workflows {
		workflowBySlug[w.Slug] = w
	}
	jobIDBySlug := map[string]string{}
	existingJobs, err := cli.ListJobs(ctx, projectID)
	if err != nil {
		return out, fmt.Errorf("list jobs for workflow resolution: %w", err)
	}
	for _, j := range existingJobs {
		jobIDBySlug[j.Slug] = j.ID
	}

	for _, res := range plan.Results {
		switch res.Kind {
		case "job":
			job := jobBySlug[res.Slug]
			switch res.Action {
			case "create":
				req := jobCreateRequest(projectID, job)
				created, err := cli.CreateJob(ctx, req, "")
				if err != nil {
					res.Error = err.Error()
					out.Failed++
				} else {
					res.ID = created.ID
					jobIDBySlug[created.Slug] = created.ID
					out.Created++
				}
			case "update":
				req := jobUpdateRequest(job)
				updated, err := cli.UpdateJob(ctx, res.ID, req)
				if err != nil {
					res.Error = err.Error()
					out.Failed++
				} else {
					res.ID = updated.ID
					jobIDBySlug[updated.Slug] = updated.ID
					out.Updated++
				}
			case "skip":
				out.Skipped++
			}
		case "workflow":
			wf := workflowBySlug[res.Slug]
			steps, err := workflowStepRequests(wf.Steps, jobIDBySlug)
			if err != nil {
				res.Error = err.Error()
				out.Failed++
				out.Results = append(out.Results, res)
				continue
			}
			switch res.Action {
			case "create":
				req := client.CreateWorkflowRequest{
					ProjectID:   projectID,
					Slug:        wf.Slug,
					Name:        firstNonEmpty(wf.Name, wf.Slug),
					Description: wf.Description,
					Enabled:     wf.Enabled,
					Steps:       steps,
				}
				created, err := cli.CreateWorkflow(ctx, req, "")
				if err != nil {
					res.Error = err.Error()
					out.Failed++
				} else {
					res.ID = created.ID
					out.Created++
				}
			case "update":
				req := workflowUpdateRequest(wf, steps)
				updated, err := cli.UpdateWorkflow(ctx, res.ID, req)
				if err != nil {
					res.Error = err.Error()
					out.Failed++
				} else {
					res.ID = updated.ID
					out.Updated++
				}
			}
		}
		out.Results = append(out.Results, res)
	}

	if prune {
		existing, err := cli.ListJobs(ctx, projectID)
		if err != nil {
			return out, fmt.Errorf("list jobs for prune: %w", err)
		}
		want := map[string]bool{}
		for _, j := range cfg.Jobs {
			want[j.Slug] = true
		}
		for _, j := range existing {
			if want[j.Slug] {
				continue
			}
			res := SyncResult{Kind: "job", Slug: j.Slug, ID: j.ID, Action: "delete"}
			if err := cli.DeleteJob(ctx, j.ID); err != nil {
				res.Error = err.Error()
				out.Failed++
			} else {
				out.Updated++
			}
			out.Results = append(out.Results, res)
		}
		existingWorkflows, err := cli.ListWorkflows(ctx, projectID)
		if err != nil {
			return out, fmt.Errorf("list workflows for prune: %w", err)
		}
		wantWorkflow := map[string]bool{}
		for _, wf := range cfg.Workflows {
			wantWorkflow[wf.Slug] = true
		}
		for _, wf := range existingWorkflows {
			if wantWorkflow[wf.Slug] {
				continue
			}
			res := SyncResult{Kind: "workflow", Slug: wf.Slug, ID: wf.ID, Action: "delete"}
			if err := cli.DeleteWorkflow(ctx, wf.ID); err != nil {
				res.Error = err.Error()
				out.Failed++
			} else {
				out.Updated++
			}
			out.Results = append(out.Results, res)
		}
	}

	return out, nil
}

func workflowStepRequests(steps []ProjectWorkflowStep, jobIDBySlug map[string]string) ([]client.WorkflowStepRequest, error) {
	out := make([]client.WorkflowStepRequest, 0, len(steps))
	for _, step := range steps {
		jobID := strings.TrimSpace(step.JobID)
		if jobID == "" && strings.TrimSpace(step.JobRef) != "" {
			var ok bool
			jobID, ok = jobIDBySlug[step.JobRef]
			if !ok {
				return nil, fmt.Errorf("workflow step %q references unknown job_ref %q", step.StepRef, step.JobRef)
			}
		}
		out = append(out, client.WorkflowStepRequest{
			JobID:     jobID,
			StepRef:   step.StepRef,
			DependsOn: step.DependsOn,
			Condition: step.Condition,
			OnFailure: step.OnFailure,
			Payload:   step.Payload,
		})
	}
	return out, nil
}

func jobCreateRequest(projectID string, job ProjectJob) client.CreateJobRequest {
	return client.CreateJobRequest{
		ProjectID:   projectID,
		Name:        firstNonEmpty(job.Name, job.Slug),
		Slug:        job.Slug,
		Description: job.Description,
		Cron:        job.Cron,
		EndpointURL: job.EndpointURL,
		MaxAttempts: job.MaxAttempts,
		TimeoutSecs: job.TimeoutSecs,
		RunTTLSecs:  job.RunTTLSecs,
		Schema:      job.Schema,
	}
}

func jobUpdateRequest(job ProjectJob) client.UpdateJobRequest {
	req := client.UpdateJobRequest{
		EndpointURL: &job.EndpointURL,
	}
	if job.Name != "" {
		req.Name = &job.Name
	}
	if job.Description != "" {
		req.Description = &job.Description
	}
	if job.Cron != "" {
		req.Cron = &job.Cron
	}
	if job.MaxAttempts != 0 {
		req.MaxAttempts = &job.MaxAttempts
	}
	if job.TimeoutSecs != 0 {
		req.TimeoutSecs = &job.TimeoutSecs
	}
	if job.RunTTLSecs != 0 {
		req.RunTTLSecs = &job.RunTTLSecs
	}
	if len(job.Schema) > 0 {
		s := job.Schema
		req.Schema = &s
	}
	return req
}

func workflowUpdateRequest(wf ProjectWorkflow, steps []client.WorkflowStepRequest) client.UpdateWorkflowRequest {
	name := firstNonEmpty(wf.Name, wf.Slug)
	description := wf.Description
	return client.UpdateWorkflowRequest{
		Name:        &name,
		Slug:        &wf.Slug,
		Description: &description,
		Enabled:     wf.Enabled,
		Steps:       &steps,
	}
}

func projectJobNeedsUpdate(job ProjectJob, existing types.Job) bool {
	if existing.EndpointURL != job.EndpointURL {
		return true
	}
	if job.Name != "" && existing.Name != job.Name {
		return true
	}
	if job.Description != "" && existing.Description != job.Description {
		return true
	}
	if job.Cron != "" && existing.Cron != job.Cron {
		return true
	}
	if job.MaxAttempts != 0 && existing.MaxAttempts != job.MaxAttempts {
		return true
	}
	if job.TimeoutSecs != 0 && existing.TimeoutSecs != job.TimeoutSecs {
		return true
	}
	if job.RunTTLSecs != 0 && existing.RunTTLSecs != job.RunTTLSecs {
		return true
	}
	return len(job.Schema) > 0 && !rawJSONEqual(job.Schema, existing.PayloadSchema)
}

func rawJSONEqual(a, b json.RawMessage) bool {
	a = compactRawJSON(a)
	b = compactRawJSON(b)
	return bytes.Equal(a, b)
}

func compactRawJSON(in json.RawMessage) json.RawMessage {
	if len(in) == 0 {
		return nil
	}
	var out bytes.Buffer
	if err := json.Compact(&out, in); err != nil {
		return in
	}
	return out.Bytes()
}

func renderSyncPlan(s *SyncSummary) {
	fmt.Fprintln(os.Stderr, styles.KeyValue("Create", fmt.Sprintf("%d", s.Created)))
	fmt.Fprintln(os.Stderr, styles.KeyValue("Update", fmt.Sprintf("%d", s.Updated)))
	fmt.Fprintln(os.Stderr, styles.KeyValue("Skip", fmt.Sprintf("%d", s.Skipped)))
	if s.Failed > 0 {
		fmt.Fprintln(os.Stderr, styles.Warn(fmt.Sprintf("Failed: %d", s.Failed)))
	}
	for _, r := range s.Results {
		line := fmt.Sprintf("  %s %s %s", r.Action, r.Kind, r.Slug)
		if r.Error != "" {
			line += " — " + r.Error
		}
		fmt.Fprintln(os.Stderr, line)
	}
}
