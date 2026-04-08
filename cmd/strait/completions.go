package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// completeJobSlugs returns a ValidArgsFunction that fetches job slugs from the API.
// Fails silently when unauthenticated or offline.
func completeJobSlugs(state *appState) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return fetchJobSlugs(state), cobra.ShellCompDirectiveNoFileComp
	}
}

// completeWorkflowSlugs returns a ValidArgsFunction that fetches workflow slugs from the API.
func completeWorkflowSlugs(state *appState) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return fetchWorkflowSlugs(state), cobra.ShellCompDirectiveNoFileComp
	}
}

func fetchJobSlugs(state *appState) []string {
	if state.opts.apiKey == "" || state.opts.projectID == "" {
		return nil
	}
	cli, err := newAPIClient(state)
	if err != nil {
		return nil
	}
	jobs, err := cli.ListJobs(nil, state.opts.projectID) //nolint:staticcheck // nil context is acceptable for completions
	if err != nil {
		return nil
	}
	slugs := make([]string, 0, len(jobs))
	for _, j := range jobs {
		slugs = append(slugs, j.Slug)
	}
	return slugs
}

func fetchWorkflowSlugs(state *appState) []string {
	if state.opts.apiKey == "" || state.opts.projectID == "" {
		return nil
	}
	cli, err := newAPIClient(state)
	if err != nil {
		return nil
	}
	workflows, err := cli.ListWorkflows(nil, state.opts.projectID) //nolint:staticcheck // nil context is acceptable for completions
	if err != nil {
		return nil
	}
	slugs := make([]string, 0, len(workflows))
	for _, wf := range workflows {
		slugs = append(slugs, wf.Slug)
	}
	return slugs
}

// completeDeploymentIDs returns a ValidArgsFunction that fetches deployment IDs
// for the job identified by the current --job flag value.
// Fails silently when unauthenticated, offline, or when no job slug is set.
func completeDeploymentIDs(state *appState, getJobSlug func() string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		slug := getJobSlug()
		if slug == "" || state.opts.projectID == "" || state.opts.apiKey == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		cli, err := newAPIClient(state)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		job, err := cli.GetJobBySlug(context.Background(), state.opts.projectID, slug)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		deps, err := cli.ListCodeDeployments(context.Background(), job.ID, 20)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		ids := make([]string, 0, len(deps))
		for _, d := range deps {
			ids = append(ids, fmt.Sprintf("%s\tv%d %s %s", d.ID, d.Version, d.Status, d.Runtime))
		}
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
}
