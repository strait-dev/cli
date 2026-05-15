package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

// DeployManifest is the on-disk shape consumed by `strait deploy push`. It is
// a fallback for projects that don't yet have strait-go SDK introspection
// wired up: write a strait.deploy.json file at the project root and the CLI
// will upsert each entry. Once strait-go v0.2.0 ships, the SDK will also be
// able to emit this same shape from its definitions registry.
type DeployManifest struct {
	Version   string           `json:"version"`
	Jobs      []DeployJob      `json:"jobs"`
	Workflows []DeployWorkflow `json:"workflows,omitempty"`
	Metadata  map[string]any   `json:"metadata,omitempty"`
}

// DeployJob is the manifest representation of a single SDK-defined job.
type DeployJob struct {
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

// DeployWorkflow is the manifest representation of a workflow.
type DeployWorkflow struct {
	Slug        string                       `json:"slug"`
	Name        string                       `json:"name,omitempty"`
	Description string                       `json:"description,omitempty"`
	Enabled     *bool                        `json:"enabled,omitempty"`
	Steps       []client.WorkflowStepRequest `json:"steps,omitempty"`
}

// DeployResult is the per-resource outcome reported by `deploy push`.
type DeployResult struct {
	Kind   string `json:"kind"`
	Slug   string `json:"slug"`
	Action string `json:"action"`
	ID     string `json:"id,omitempty"`
	Error  string `json:"error,omitempty"`
}

// DeploySummary aggregates DeployResult across a push run.
type DeploySummary struct {
	Created int            `json:"created"`
	Updated int            `json:"updated"`
	Skipped int            `json:"skipped"`
	Failed  int            `json:"failed"`
	Results []DeployResult `json:"results"`
}

func newDeployCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Push SDK-defined jobs and workflows to the orchestrator",
	}
	cmd.AddCommand(newDeployPushCommand(state))
	return cmd
}

func newDeployPushCommand(state *appState) *cobra.Command {
	var dir string
	var manifestPath string
	var dryRun bool
	var prune bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Upsert jobs and workflows defined in strait.deploy.json",
		Long: `Reads SDK definitions from a strait.deploy.json manifest at --dir (or
--manifest) and upserts each job/workflow against the orchestrator.

Until strait-go v0.2.0 ships, this is the manifest-based push path. Once
the SDK is published, this command will additionally auto-discover
definitions by introspecting the user's TS or Go project.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectID := state.opts.projectID
			if strings.TrimSpace(projectID) == "" {
				return fmt.Errorf("project is required (set --project, STRAIT_PROJECT, or context)")
			}

			path := manifestPath
			if path == "" {
				root := dir
				if root == "" {
					root = "."
				}
				path = filepath.Join(root, "strait.deploy.json")
			}
			manifest, err := loadDeployManifest(path)
			if err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			summary, err := planDeploy(cmd.Context(), cli, projectID, manifest)
			if err != nil {
				return err
			}

			if dryRun {
				if isTTYRich(state) {
					renderDeployPlan(summary)
					return nil
				}
				return printData(state, summary)
			}

			if prune {
				if err := requireConfirmation(state, "Pruning removes jobs not in the manifest. Continue?", yes); err != nil {
					return err
				}
			}

			final, err := applyDeploy(cmd.Context(), cli, projectID, manifest, summary, prune)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				renderDeployPlan(final)
				if final.Failed > 0 {
					return fmt.Errorf("%d resource(s) failed to deploy", final.Failed)
				}
				return nil
			}
			if err := printData(state, final); err != nil {
				return err
			}
			if final.Failed > 0 {
				return fmt.Errorf("%d resource(s) failed to deploy", final.Failed)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "project root containing strait.deploy.json (default: current dir)")
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "explicit path to a manifest file (overrides --dir)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the deploy plan without making any changes")
	cmd.Flags().BoolVar(&prune, "prune", false, "delete jobs not present in the manifest (destructive)")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt for --prune")
	return cmd
}

// loadDeployManifest reads and validates a manifest from disk.
func loadDeployManifest(path string) (*DeployManifest, error) {
	abs := path
	if !filepath.IsAbs(abs) {
		cwd, _ := os.Getwd()
		abs = filepath.Join(cwd, abs)
	}
	data, err := os.ReadFile(abs) //nolint:gosec // path is user-provided via flag
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("manifest not found at %s; run `strait deploy push --manifest <path>` or create the file", path)
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m DeployManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest JSON: %w", err)
	}
	if len(m.Jobs) == 0 && len(m.Workflows) == 0 {
		return nil, fmt.Errorf("manifest %s has no jobs or workflows", path)
	}
	for i, j := range m.Jobs {
		if strings.TrimSpace(j.Slug) == "" {
			return nil, fmt.Errorf("manifest jobs[%d]: slug is required", i)
		}
		if strings.TrimSpace(j.EndpointURL) == "" {
			return nil, fmt.Errorf("manifest jobs[%d] (%s): endpoint_url is required", i, j.Slug)
		}
	}
	for i, w := range m.Workflows {
		if strings.TrimSpace(w.Slug) == "" {
			return nil, fmt.Errorf("manifest workflows[%d]: slug is required", i)
		}
	}
	return &m, nil
}

// planDeploy classifies each manifest entry as create / update / skip without
// applying any changes.
func planDeploy(ctx context.Context, cli *client.Client, projectID string, manifest *DeployManifest) (*DeploySummary, error) {
	existing, err := cli.ListJobs(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list existing jobs: %w", err)
	}
	bySlug := make(map[string]string, len(existing))
	endpointBySlug := make(map[string]string, len(existing))
	for _, j := range existing {
		bySlug[j.Slug] = j.ID
		endpointBySlug[j.Slug] = j.EndpointURL
	}

	summary := &DeploySummary{}
	for _, job := range manifest.Jobs {
		res := DeployResult{Kind: "job", Slug: job.Slug}
		id, ok := bySlug[job.Slug]
		switch {
		case !ok:
			res.Action = "create"
			summary.Created++
		case endpointBySlug[job.Slug] == job.EndpointURL:
			res.Action = "skip"
			res.ID = id
			summary.Skipped++
		default:
			res.Action = "update"
			res.ID = id
			summary.Updated++
		}
		summary.Results = append(summary.Results, res)
	}

	for _, wf := range manifest.Workflows {
		res := DeployResult{Kind: "workflow", Slug: wf.Slug, Action: "create"}
		summary.Created++
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

// applyDeploy executes the create/update operations recorded in summary.
// Workflows are always created via CreateWorkflow (the API treats it as
// idempotent by slug); failures are recorded per-result rather than aborting.
func applyDeploy(ctx context.Context, cli *client.Client, projectID string, manifest *DeployManifest, plan *DeploySummary, prune bool) (*DeploySummary, error) {
	out := &DeploySummary{}
	jobByID := map[string]DeployJob{}
	for _, j := range manifest.Jobs {
		jobByID[j.Slug] = j
	}

	for _, res := range plan.Results {
		switch res.Kind {
		case "job":
			job := jobByID[res.Slug]
			switch res.Action {
			case "create":
				req := jobCreateRequest(projectID, job)
				created, err := cli.CreateJob(ctx, req, "")
				if err != nil {
					res.Error = err.Error()
					out.Failed++
				} else {
					res.ID = created.ID
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
					out.Updated++
				}
			case "skip":
				out.Skipped++
			}
		case "workflow":
			wf := findWorkflow(manifest.Workflows, res.Slug)
			req := client.CreateWorkflowRequest{
				ProjectID:   projectID,
				Slug:        wf.Slug,
				Name:        wf.Name,
				Description: wf.Description,
				Enabled:     wf.Enabled,
				Steps:       wf.Steps,
			}
			created, err := cli.CreateWorkflow(ctx, req, "")
			if err != nil {
				res.Error = err.Error()
				out.Failed++
			} else {
				res.ID = created.ID
				out.Created++
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
		for _, j := range manifest.Jobs {
			want[j.Slug] = true
		}
		for _, j := range existing {
			if want[j.Slug] {
				continue
			}
			res := DeployResult{Kind: "job", Slug: j.Slug, ID: j.ID, Action: "delete"}
			if err := cli.DeleteJob(ctx, j.ID); err != nil {
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

func jobCreateRequest(projectID string, job DeployJob) client.CreateJobRequest {
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

func jobUpdateRequest(job DeployJob) client.UpdateJobRequest {
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

func findWorkflow(workflows []DeployWorkflow, slug string) DeployWorkflow {
	for _, w := range workflows {
		if w.Slug == slug {
			return w
		}
	}
	return DeployWorkflow{}
}

func renderDeployPlan(s *DeploySummary) {
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
