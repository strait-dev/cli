package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// This file adds cloud analytics client methods. All endpoints are read-only
// GET requests that return raw JSON so the CLI can render the server payload
// directly. Paths are literal so the OpenAPI contract test validates them.
// Most endpoints accept a project_id query param and an optional time window
// (from/to as RFC3339 strings). Endpoints with path parameters take those ids
// as explicit arguments and build the path with joinPath.

func analyticsGet(ctx context.Context, c *Client, endpoint string, query url.Values) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func buildAnalyticsQuery(projectID, from, to string) url.Values {
	q := url.Values{}
	q.Set("project_id", projectID)
	if from != "" {
		q.Set("from", from)
	}
	if to != "" {
		q.Set("to", to)
	}
	return q
}

// GetAnalyticsApprovals returns approval analytics for a project.
func (c *Client) GetAnalyticsApprovals(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/approvals", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsCostInsights returns cost insight analytics for a project.
func (c *Client) GetAnalyticsCostInsights(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/cost-insights", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsCostsTrends returns cost trend analytics for a project.
func (c *Client) GetAnalyticsCostsTrends(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/costs/trends", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsCostsTop returns top-cost analytics for a project.
func (c *Client) GetAnalyticsCostsTop(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/costs/top", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsCostsByTrigger returns cost-by-trigger analytics for a project.
func (c *Client) GetAnalyticsCostsByTrigger(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/costs/by-trigger", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsCostsForecast returns cost forecast analytics for a project.
func (c *Client) GetAnalyticsCostsForecast(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/costs/forecast", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsRunsTimeline returns run timeline analytics for a project.
func (c *Client) GetAnalyticsRunsTimeline(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/runs/timeline", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsRunsDurationDistribution returns run duration distribution analytics for a project.
func (c *Client) GetAnalyticsRunsDurationDistribution(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/runs/duration-distribution", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsRunsFailureReasons returns run failure reason analytics for a project.
func (c *Client) GetAnalyticsRunsFailureReasons(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/runs/failure-reasons", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsRunsSummary returns run summary analytics for a project.
func (c *Client) GetAnalyticsRunsSummary(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/runs/summary", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsRunsByTrigger returns run-by-trigger analytics for a project.
func (c *Client) GetAnalyticsRunsByTrigger(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/runs/by-trigger", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsJobsComparison returns job comparison analytics for a project.
func (c *Client) GetAnalyticsJobsComparison(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/jobs/comparison", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsJobsByVersion returns job-by-version analytics for a project.
func (c *Client) GetAnalyticsJobsByVersion(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/jobs/by-version", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsJobsCostRanking returns job cost-ranking analytics for a project.
func (c *Client) GetAnalyticsJobsCostRanking(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/jobs/cost-ranking", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsJobHistory returns run history analytics for a specific job.
func (c *Client) GetAnalyticsJobHistory(ctx context.Context, jobID, projectID, from, to string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/analytics/jobs", jobID, "history")
	if err != nil {
		return nil, fmt.Errorf("invalid job id: %w", err)
	}
	return analyticsGet(ctx, c, endpoint, buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsTagsSummary returns tag summary analytics for a project.
func (c *Client) GetAnalyticsTagsSummary(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/tags/summary", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsTagsTopFailing returns top-failing tag analytics for a project.
func (c *Client) GetAnalyticsTagsTopFailing(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/tags/top-failing", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsTagsCost returns tag cost analytics for a project.
func (c *Client) GetAnalyticsTagsCost(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/tags/cost", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsWebhooksDeliveryStats returns webhook delivery stats for a project.
func (c *Client) GetAnalyticsWebhooksDeliveryStats(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/webhooks/delivery-stats", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsWebhooksEndpointHealth returns webhook endpoint health analytics for a project.
func (c *Client) GetAnalyticsWebhooksEndpointHealth(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/webhooks/endpoint-health", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsWebhooksTopFailing returns top-failing webhook analytics for a project.
func (c *Client) GetAnalyticsWebhooksTopFailing(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/webhooks/top-failing", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsWorkflowsCompletionRates returns workflow completion rate analytics for a project.
func (c *Client) GetAnalyticsWorkflowsCompletionRates(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/workflows/completion-rates", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsWorkflowsSummary returns workflow summary analytics for a project.
func (c *Client) GetAnalyticsWorkflowsSummary(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/workflows/summary", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsWorkflowStepDurations returns step duration analytics for a specific workflow.
func (c *Client) GetAnalyticsWorkflowStepDurations(ctx context.Context, workflowID, projectID, from, to string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/analytics/workflows", workflowID, "step-durations")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow id: %w", err)
	}
	return analyticsGet(ctx, c, endpoint, buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsEventsVolume returns event volume analytics for a project.
func (c *Client) GetAnalyticsEventsVolume(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/events/volume", buildAnalyticsQuery(projectID, from, to))
}

// GetAnalyticsEventsLatency returns event latency analytics for a project.
func (c *Client) GetAnalyticsEventsLatency(ctx context.Context, projectID, from, to string) (json.RawMessage, error) {
	return analyticsGet(ctx, c, "/v1/analytics/events/latency", buildAnalyticsQuery(projectID, from, to))
}
