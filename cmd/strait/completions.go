package main

import (
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
