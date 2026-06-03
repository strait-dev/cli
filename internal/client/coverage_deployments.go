package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// This file adds deployment/canary client methods. Reads return raw JSON so the
// CLI can render the server payload directly without mirroring every response
// schema; writes accept explicit fields or a raw manifest body. Paths are
// literal so the OpenAPI contract test validates them.

// ListDeployments returns deployment versions, optionally filtered by environment.
func (c *Client) ListDeployments(ctx context.Context, environment string, limit int) (json.RawMessage, error) {
	query := url.Values{}
	if environment != "" {
		query.Set("environment", environment)
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	var out json.RawMessage
	if err := c.doListJSON(ctx, "/v1/deployments", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateDeployment creates a deployment version from a raw JSON body matching the
// server's CreateDeploymentRequest schema.
func (c *Client) CreateDeployment(ctx context.Context, body json.RawMessage) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/deployments", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) deploymentAction(ctx context.Context, deploymentID, action, projectID, environment string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/deployments", deploymentID, action)
	if err != nil {
		return nil, fmt.Errorf("invalid deployment id: %w", err)
	}
	body := map[string]any{"project_id": projectID, "environment": environment}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FinalizeDeployment finalizes a deployment version.
func (c *Client) FinalizeDeployment(ctx context.Context, deploymentID, projectID, environment string) (json.RawMessage, error) {
	return c.deploymentAction(ctx, deploymentID, "finalize", projectID, environment)
}

// PromoteDeployment promotes a deployment version to active.
func (c *Client) PromoteDeployment(ctx context.Context, deploymentID, projectID, environment string) (json.RawMessage, error) {
	return c.deploymentAction(ctx, deploymentID, "promote", projectID, environment)
}

// RollbackDeployment rolls back to a previous deployment version.
func (c *Client) RollbackDeployment(ctx context.Context, deploymentID, projectID, environment string) (json.RawMessage, error) {
	return c.deploymentAction(ctx, deploymentID, "rollback", projectID, environment)
}

// CreateCanaryDeployment creates a canary deployment from a raw JSON body
// matching the server's CreateCanaryDeploymentRequest schema.
func (c *Client) CreateCanaryDeployment(ctx context.Context, body json.RawMessage) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/canary-deployments", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}
