package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// This file adds billing client methods. Reads return raw JSON so the CLI can
// render the server payload directly without mirroring every response schema.
// Paths are literal so the OpenAPI contract test validates them.

// GetSpendingLimit returns the org-level spending limit.
func (c *Client) GetSpendingLimit(ctx context.Context, orgID string) (json.RawMessage, error) {
	query := url.Values{}
	if orgID != "" {
		query.Set("org_id", orgID)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/spending-limit", query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SetSpendingLimit upserts the org-level spending limit.
func (c *Client) SetSpendingLimit(ctx context.Context, orgID string, limitMicroUSD int64, action string) (json.RawMessage, error) {
	query := url.Values{}
	if orgID != "" {
		query.Set("org_id", orgID)
	}
	body := map[string]any{
		"limit_microusd": limitMicroUSD,
		"action":         action,
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPut, "/v1/spending-limit", query, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetProjectBudget returns the project-level budget.
func (c *Client) GetProjectBudget(ctx context.Context, projectID string) (json.RawMessage, error) {
	query := url.Values{}
	if projectID != "" {
		query.Set("project_id", projectID)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/project-budget", query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SetProjectBudget upserts the project-level budget.
func (c *Client) SetProjectBudget(ctx context.Context, projectID string, budgetMicroUSD int64, action string) (json.RawMessage, error) {
	body := map[string]any{
		"project_id":      projectID,
		"budget_microusd": budgetMicroUSD,
		"action":          action,
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPut, "/v1/project-budget", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAnomalyConfig returns the org-level anomaly detection configuration.
func (c *Client) GetAnomalyConfig(ctx context.Context, orgID string) (json.RawMessage, error) {
	query := url.Values{}
	if orgID != "" {
		query.Set("org_id", orgID)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/anomaly-config", query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SetAnomalyConfig upserts the org-level anomaly detection thresholds.
func (c *Client) SetAnomalyConfig(ctx context.Context, orgID string, warning, critical float64) (json.RawMessage, error) {
	query := url.Values{}
	if orgID != "" {
		query.Set("org_id", orgID)
	}
	body := map[string]any{
		"warning_threshold":  warning,
		"critical_threshold": critical,
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPut, "/v1/anomaly-config", query, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListRegions returns all available regions.
func (c *Client) ListRegions(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/regions", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetDowngradePreview returns a preview of the effects of downgrading an org to a target tier.
func (c *Client) GetDowngradePreview(ctx context.Context, orgID, targetTier string) (json.RawMessage, error) {
	query := url.Values{}
	if orgID != "" {
		query.Set("org_id", orgID)
	}
	if targetTier != "" {
		query.Set("target_tier", targetTier)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/downgrade-preview", query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CheckOrgLimit checks whether a user has reached the org limit for a given plan tier.
func (c *Client) CheckOrgLimit(ctx context.Context, userID, planTier string) (json.RawMessage, error) {
	query := url.Values{}
	if userID != "" {
		query.Set("user_id", userID)
	}
	if planTier != "" {
		query.Set("plan_tier", planTier)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/billing/check-org-limit", query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
