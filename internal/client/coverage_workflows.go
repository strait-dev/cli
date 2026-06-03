package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// This file adds workflow graph/version/canary and workflow-run analysis client
// methods. Reads return raw JSON so the CLI can render the server payload
// directly without mirroring every response schema. Paths are literal so the
// OpenAPI contract test validates them.

// GetWorkflowGraph returns the execution graph for a workflow.
func (c *Client) GetWorkflowGraph(ctx context.Context, workflowID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflows", workflowID, "graph")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflowActiveVersions returns the active versions for a workflow.
func (c *Client) GetWorkflowActiveVersions(ctx context.Context, workflowID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflows", workflowID, "active-versions")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflowVersion returns a specific version of a workflow.
func (c *Client) GetWorkflowVersion(ctx context.Context, workflowID, versionID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflows", workflowID, "versions", versionID)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow or version id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflowVersionImpact returns the impact analysis for a specific workflow version.
func (c *Client) GetWorkflowVersionImpact(ctx context.Context, workflowID, versionID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflows", workflowID, "versions", versionID, "impact")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow or version id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflowVersionSteps returns the steps for a specific workflow version.
func (c *Client) GetWorkflowVersionSteps(ctx context.Context, workflowID, versionID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflows", workflowID, "versions", versionID, "steps")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow or version id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflowCanary returns the canary configuration for a workflow.
func (c *Client) GetWorkflowCanary(ctx context.Context, workflowID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflows", workflowID, "canary")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SetWorkflowCanary sets the canary traffic percentage for a workflow.
func (c *Client) SetWorkflowCanary(ctx context.Context, workflowID string, trafficPct float64) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflows", workflowID, "canary")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow id: %w", err)
	}
	body := map[string]float64{"traffic_pct": trafficPct}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPatch, endpoint, nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RollbackWorkflowCanary rolls back the canary for a workflow.
func (c *Client) RollbackWorkflowCanary(ctx context.Context, workflowID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflows", workflowID, "canary", "rollback")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CompareWorkflowRuns returns a comparison between two workflow runs.
func (c *Client) CompareWorkflowRuns(ctx context.Context, workflowRunID, otherRunID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflow-runs", workflowRunID, "compare", otherRunID)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflowRunCompensationPlan returns the compensation plan for a workflow run.
func (c *Client) GetWorkflowRunCompensationPlan(ctx context.Context, workflowRunID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflow-runs", workflowRunID, "compensation-plan")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflowRunDebug returns debug information for a workflow run.
func (c *Client) GetWorkflowRunDebug(ctx context.Context, workflowRunID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflow-runs", workflowRunID, "debug")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflowRunExplain returns the explanation for a workflow run.
func (c *Client) GetWorkflowRunExplain(ctx context.Context, workflowRunID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflow-runs", workflowRunID, "explain")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflowRunGraph returns the execution graph for a workflow run.
func (c *Client) GetWorkflowRunGraph(ctx context.Context, workflowRunID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflow-runs", workflowRunID, "graph")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflowRunLabels returns the labels for a workflow run.
func (c *Client) GetWorkflowRunLabels(ctx context.Context, workflowRunID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflow-runs", workflowRunID, "labels")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflowRunTimeline returns the timeline for a workflow run.
func (c *Client) GetWorkflowRunTimeline(ctx context.Context, workflowRunID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflow-runs", workflowRunID, "timeline")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CompensateWorkflowRun triggers compensation for a workflow run.
func (c *Client) CompensateWorkflowRun(ctx context.Context, workflowRunID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflow-runs", workflowRunID, "compensate")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ReplayWorkflowStepSubtree replays a step subtree within a workflow run.
func (c *Client) ReplayWorkflowStepSubtree(ctx context.Context, runID, stepRef string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/workflow-runs", runID, "steps", stepRef, "replay-subtree")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow run or step ref: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// BulkCancelWorkflowRuns cancels multiple workflow runs.
func (c *Client) BulkCancelWorkflowRuns(ctx context.Context, workflowRunIDs []string) (json.RawMessage, error) {
	body := map[string][]string{"workflow_run_ids": workflowRunIDs}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/workflow-runs/bulk-cancel", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// BulkReplayWorkflowRuns replays multiple workflow runs.
func (c *Client) BulkReplayWorkflowRuns(ctx context.Context, workflowRunIDs []string) (json.RawMessage, error) {
	body := map[string][]string{"workflow_run_ids": workflowRunIDs}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/workflow-runs/bulk-replay", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}
