package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// This file adds RBAC roles, tag-policies, and extended usage client methods.
// Reads return raw JSON so the CLI can render the server payload directly
// without mirroring every response schema. Paths are literal so the OpenAPI
// contract test validates them.

// ListRolesRaw returns all roles as raw JSON using the list envelope.
func (c *Client) ListRolesRaw(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doListJSON(ctx, "/v1/roles", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetRole returns a single role by ID.
func (c *Client) GetRole(ctx context.Context, roleID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/roles", roleID)
	if err != nil {
		return nil, fmt.Errorf("invalid role id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RoleRequest is the request body for creating or updating a role.
type RoleRequest struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Permissions  []string `json:"permissions"`
	ParentRoleID string   `json:"parent_role_id,omitempty"`
}

// CreateRole creates a new role.
func (c *Client) CreateRole(ctx context.Context, req RoleRequest) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/roles", nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateRole updates an existing role by ID.
func (c *Client) UpdateRole(ctx context.Context, roleID string, req RoleRequest) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/roles", roleID)
	if err != nil {
		return nil, fmt.Errorf("invalid role id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPatch, endpoint, nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteRoleRaw deletes a role by ID.
func (c *Client) DeleteRoleRaw(ctx context.Context, roleID string) error {
	endpoint, err := joinPath("/v1/roles", roleID)
	if err != nil {
		return fmt.Errorf("invalid role id: %w", err)
	}
	var discard json.RawMessage
	return c.doJSON(ctx, http.MethodDelete, endpoint, nil, nil, &discard)
}

// ListTagPolicies returns all tag policies as raw JSON using the list envelope.
func (c *Client) ListTagPolicies(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doListJSON(ctx, "/v1/tag-policies", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// TagPolicyRequest is the request body for creating a tag policy.
type TagPolicyRequest struct {
	ProjectID    string   `json:"project_id"`
	ResourceType string   `json:"resource_type"`
	UserID       string   `json:"user_id"`
	TagKey       string   `json:"tag_key"`
	TagValue     string   `json:"tag_value,omitempty"`
	Actions      []string `json:"actions"`
}

// CreateTagPolicy creates a new tag policy.
func (c *Client) CreateTagPolicy(ctx context.Context, req TagPolicyRequest) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/tag-policies", nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteTagPolicy deletes a tag policy by ID.
func (c *Client) DeleteTagPolicy(ctx context.Context, policyID string) error {
	endpoint, err := joinPath("/v1/tag-policies", policyID)
	if err != nil {
		return fmt.Errorf("invalid policy id: %w", err)
	}
	var discard json.RawMessage
	return c.doJSON(ctx, http.MethodDelete, endpoint, nil, nil, &discard)
}

// GetUsageAnomalies returns usage anomaly data.
func (c *Client) GetUsageAnomalies(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/usage/anomalies", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetUsageByProject returns usage broken down by project.
func (c *Client) GetUsageByProject(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/usage/projects", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ExportUsage exports usage data in the requested format.
func (c *Client) ExportUsage(ctx context.Context, format string) (json.RawMessage, error) {
	query := url.Values{}
	if format != "" {
		query.Set("format", format)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/usage/export", query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetUsageEmailPreferences returns the usage email preferences for the given org.
func (c *Client) GetUsageEmailPreferences(ctx context.Context, orgID string) (json.RawMessage, error) {
	query := url.Values{}
	if orgID != "" {
		query.Set("org_id", orgID)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/v1/usage/email-preferences", query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// setUsageEmailPreferencesBody holds the fields for the email-preferences PUT request.
type setUsageEmailPreferencesBody struct {
	MonthlyUsageEmail bool `json:"monthly_usage_email"`
}

// SetUsageEmailPreferences sets the usage email preferences for the given org.
func (c *Client) SetUsageEmailPreferences(ctx context.Context, orgID string, monthlyUsageEmail bool) (json.RawMessage, error) {
	query := url.Values{}
	if orgID != "" {
		query.Set("org_id", orgID)
	}
	body := setUsageEmailPreferencesBody{
		MonthlyUsageEmail: monthlyUsageEmail,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal email-preferences body: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPut, "/v1/usage/email-preferences", query, json.RawMessage(encoded), &out); err != nil {
		return nil, err
	}
	return out, nil
}
