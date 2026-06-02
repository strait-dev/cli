package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// This file adds extended run client methods. Reads return raw JSON so the
// CLI can render the server payload directly without mirroring every response
// schema; writes accept explicit fields. Paths are literal so the OpenAPI
// contract test validates them.

// ListRunChildren returns the child runs spawned by the given run.
func (c *Client) ListRunChildren(ctx context.Context, runID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/runs", runID, "children")
	if err != nil {
		return nil, fmt.Errorf("invalid run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doListJSON(ctx, endpoint, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetRunLineage returns the lineage graph for the given run.
func (c *Client) GetRunLineage(ctx context.Context, runID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/runs", runID, "lineage")
	if err != nil {
		return nil, fmt.Errorf("invalid run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetRunDependencyStatus returns the dependency status for the given run.
func (c *Client) GetRunDependencyStatus(ctx context.Context, runID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/runs", runID, "dependency-status")
	if err != nil {
		return nil, fmt.Errorf("invalid run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListRunState returns the state entries for the given run.
func (c *Client) ListRunState(ctx context.Context, runID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/runs", runID, "state")
	if err != nil {
		return nil, fmt.Errorf("invalid run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doListJSON(ctx, endpoint, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListRunResources returns the resources associated with the given run.
func (c *Client) ListRunResources(ctx context.Context, runID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/runs", runID, "resources")
	if err != nil {
		return nil, fmt.Errorf("invalid run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doListJSON(ctx, endpoint, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetRunDebugBundle returns the debug bundle for the given run.
func (c *Client) GetRunDebugBundle(ctx context.Context, runID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/runs", runID, "debug-bundle")
	if err != nil {
		return nil, fmt.Errorf("invalid run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListRunUsage returns the usage records for the given run.
func (c *Client) ListRunUsage(ctx context.Context, runID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/runs", runID, "usage")
	if err != nil {
		return nil, fmt.Errorf("invalid run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doListJSON(ctx, endpoint, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RestartRun restarts the given run.
func (c *Client) RestartRun(ctx context.Context, runID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/runs", runID, "restart")
	if err != nil {
		return nil, fmt.Errorf("invalid run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PauseRun pauses the given run.
func (c *Client) PauseRun(ctx context.Context, runID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/runs", runID, "pause")
	if err != nil {
		return nil, fmt.Errorf("invalid run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ResumeRun resumes the given run.
func (c *Client) ResumeRun(ctx context.Context, runID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/runs", runID, "resume")
	if err != nil {
		return nil, fmt.Errorf("invalid run id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SetRunDebug enables or disables debug mode on the given run.
func (c *Client) SetRunDebug(ctx context.Context, runID string, enabled bool) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/runs", runID, "debug")
	if err != nil {
		return nil, fmt.Errorf("invalid run id: %w", err)
	}
	body := map[string]any{"debug_mode": enabled}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ResetRunIdempotencyKey deletes the idempotency key for the given run.
func (c *Client) ResetRunIdempotencyKey(ctx context.Context, runID string) error {
	endpoint, err := joinPath("/v1/runs", runID, "idempotency-key")
	if err != nil {
		return fmt.Errorf("invalid run id: %w", err)
	}
	return c.doJSON(ctx, http.MethodDelete, endpoint, nil, nil, nil)
}

// BulkCancelRunsByIDs cancels the runs identified by the given IDs.
func (c *Client) BulkCancelRunsByIDs(ctx context.Context, runIDs []string) (json.RawMessage, error) {
	body := map[string]any{"run_ids": runIDs}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/runs/bulk-cancel", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// BulkCancelAllRuns cancels all runs matching the given optional filters.
func (c *Client) BulkCancelAllRuns(ctx context.Context, jobID, status, batchID, triggeredBy string) (json.RawMessage, error) {
	body := map[string]any{}
	if jobID != "" {
		body["job_id"] = jobID
	}
	if status != "" {
		body["status"] = status
	}
	if batchID != "" {
		body["batch_id"] = batchID
	}
	if triggeredBy != "" {
		body["triggered_by"] = triggeredBy
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/runs/bulk-cancel-all", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// BulkReplayRuns replays the runs identified by the given IDs.
func (c *Client) BulkReplayRuns(ctx context.Context, runIDs []string) (json.RawMessage, error) {
	body := map[string]any{"run_ids": runIDs}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/runs/bulk-replay", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// BulkDLQReplayRuns replays dead-letter-queue runs by ID.
func (c *Client) BulkDLQReplayRuns(ctx context.Context, runIDs []string, projectID string, limit int) (json.RawMessage, error) {
	body := map[string]any{
		"run_ids":    runIDs,
		"project_id": projectID,
		"limit":      limit,
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/runs/bulk-dlq-replay", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}
