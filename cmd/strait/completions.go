package main

import (
	"context"

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

// completeEnvironmentSlugs returns a ValidArgsFunction that fetches environment slugs from the API.
func completeEnvironmentSlugs(state *appState) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if state.opts.apiKey == "" || state.opts.projectID == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		cli, err := newAPIClient(state)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		envs, err := cli.ListEnvironments(context.Background(), state.opts.projectID)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		slugs := make([]string, 0, len(envs))
		for _, e := range envs {
			slugs = append(slugs, e.Slug)
		}
		return slugs, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeWebhookIDs returns a ValidArgsFunction that fetches webhook IDs from the API.
func completeWebhookIDs(state *appState) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if state.opts.apiKey == "" || state.opts.projectID == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		cli, err := newAPIClient(state)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		hooks, err := cli.ListWebhooks(context.Background(), state.opts.projectID)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		ids := make([]string, 0, len(hooks))
		for _, h := range hooks {
			ids = append(ids, h.ID)
		}
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeEventSourceSlugs returns a ValidArgsFunction that fetches event source slugs from the API.
func completeEventSourceSlugs(state *appState) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if state.opts.apiKey == "" || state.opts.projectID == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		cli, err := newAPIClient(state)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		sources, err := cli.ListEventSources(context.Background(), state.opts.projectID)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		slugs := make([]string, 0, len(sources))
		for _, s := range sources {
			slugs = append(slugs, s.Slug)
		}
		return slugs, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeJobGroupSlugs returns a ValidArgsFunction that fetches job group slugs from the API.
func completeJobGroupSlugs(state *appState) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if state.opts.apiKey == "" || state.opts.projectID == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		cli, err := newAPIClient(state)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		groups, err := cli.ListJobGroups(context.Background(), state.opts.projectID)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		slugs := make([]string, 0, len(groups))
		for _, g := range groups {
			slugs = append(slugs, g.Slug)
		}
		return slugs, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeNotificationChannelIDs returns a ValidArgsFunction for notification channel IDs.
func completeNotificationChannelIDs(state *appState) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if state.opts.apiKey == "" || state.opts.projectID == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		cli, err := newAPIClient(state)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		channels, err := cli.ListNotificationChannels(context.Background(), state.opts.projectID)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		ids := make([]string, 0, len(channels))
		for _, c := range channels {
			ids = append(ids, c.ID)
		}
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeLogDrainIDs returns a ValidArgsFunction for log drain IDs.
func completeLogDrainIDs(state *appState) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if state.opts.apiKey == "" || state.opts.projectID == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		cli, err := newAPIClient(state)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		drains, err := cli.ListLogDrains(context.Background(), state.opts.projectID)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		ids := make([]string, 0, len(drains))
		for _, d := range drains {
			ids = append(ids, d.ID)
		}
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
}
