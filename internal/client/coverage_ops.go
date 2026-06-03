package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// This file adds export, stats, batch-operations, and organizations client
// methods. Reads return raw JSON so the CLI can render the server payload
// directly without mirroring every response schema. Paths are literal so the
// OpenAPI contract test validates them.

// ExportJobs returns exported job data in the requested format.
func (c *Client) ExportJobs(ctx context.Context, format string) (json.RawMessage, error) {
	query := url.Values{}
	if format != "" {
		query.Set("format", format)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/export/jobs", query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ExportRuns returns exported run data in the requested format, optionally
// filtered by time range.
func (c *Client) ExportRuns(ctx context.Context, format, from, to string) (json.RawMessage, error) {
	query := url.Values{}
	if format != "" {
		query.Set("format", format)
	}
	if from != "" {
		query.Set("from", from)
	}
	if to != "" {
		query.Set("to", to)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/export/runs", query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ExportWorkflows returns exported workflow data in the requested format.
func (c *Client) ExportWorkflows(ctx context.Context, format string) (json.RawMessage, error) {
	query := url.Values{}
	if format != "" {
		query.Set("format", format)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/export/workflows", query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetStats returns the account/project statistics summary.
func (c *Client) GetStats(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/stats", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListBatchOperations returns a list of batch operations, optionally limited.
func (c *Client) ListBatchOperations(ctx context.Context, limit int) (json.RawMessage, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	var out json.RawMessage
	if err := c.doListJSON(ctx, "/v1/batch-operations", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetBatchOperation returns a single batch operation by ID.
func (c *Client) GetBatchOperation(ctx context.Context, batchID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/batch-operations", batchID)
	if err != nil {
		return nil, fmt.Errorf("invalid batch operation id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListOrgJobs returns a list of jobs belonging to the given organization.
func (c *Client) ListOrgJobs(ctx context.Context, orgID string, limit int) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/organizations", orgID, "jobs")
	if err != nil {
		return nil, fmt.Errorf("invalid organization id: %w", err)
	}
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	var out json.RawMessage
	if err := c.doListJSON(ctx, endpoint, query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListOrgRuns returns a list of runs belonging to the given organization.
func (c *Client) ListOrgRuns(ctx context.Context, orgID string, limit int) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/organizations", orgID, "runs")
	if err != nil {
		return nil, fmt.Errorf("invalid organization id: %w", err)
	}
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	var out json.RawMessage
	if err := c.doListJSON(ctx, endpoint, query, &out); err != nil {
		return nil, err
	}
	return out, nil
}
