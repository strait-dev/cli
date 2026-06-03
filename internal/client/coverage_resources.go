package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// This file adds job, event-source, secret, notification, and api-key client
// methods that were not covered by the initial implementation. Reads return raw
// JSON so the CLI can render the server payload directly without mirroring
// every response schema. Paths are literal so the OpenAPI contract test
// validates them.

// PauseJob pauses a job by ID.
func (c *Client) PauseJob(ctx context.Context, jobID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/jobs", jobID, "pause")
	if err != nil {
		return nil, fmt.Errorf("invalid job id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ResumeJob resumes a paused job by ID.
func (c *Client) ResumeJob(ctx context.Context, jobID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/jobs", jobID, "resume")
	if err != nil {
		return nil, fmt.Errorf("invalid job id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// BatchEnableJobs enables multiple jobs by ID.
func (c *Client) BatchEnableJobs(ctx context.Context, ids []string) (json.RawMessage, error) {
	body := map[string]any{"ids": ids}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/jobs/batch-enable", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// BatchDisableJobs disables multiple jobs by ID.
func (c *Client) BatchDisableJobs(ctx context.Context, ids []string) (json.RawMessage, error) {
	body := map[string]any{"ids": ids}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/jobs/batch-disable", nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// BulkTriggerJobRaw triggers multiple job runs at once using a raw JSON body.
// The body must contain {"items":[...]} per the server schema.
func (c *Client) BulkTriggerJobRaw(ctx context.Context, jobID string, body json.RawMessage) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/jobs", jobID, "trigger", "bulk")
	if err != nil {
		return nil, fmt.Errorf("invalid job id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetJobVersion returns a specific version of a job.
func (c *Client) GetJobVersion(ctx context.Context, jobID, versionID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/jobs", jobID, "versions", versionID)
	if err != nil {
		return nil, fmt.Errorf("invalid job or version id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteJobDependency removes a dependency edge from a job.
func (c *Client) DeleteJobDependency(ctx context.Context, jobID, depID string) error {
	endpoint, err := joinPath("/v1/jobs", jobID, "dependencies", depID)
	if err != nil {
		return fmt.Errorf("invalid job or dependency id: %w", err)
	}
	return c.doJSON(ctx, http.MethodDelete, endpoint, nil, nil, nil)
}

// ListEventSourceSubscriptions returns subscriptions for an event source.
func (c *Client) ListEventSourceSubscriptions(ctx context.Context, sourceID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/event-sources", sourceID, "subscriptions")
	if err != nil {
		return nil, fmt.Errorf("invalid event source id: %w", err)
	}
	var out json.RawMessage
	if err := c.doListJSON(ctx, endpoint, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SubscribeEventSourceRequest is the request body for SubscribeEventSource.
type SubscribeEventSourceRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Enabled    *bool  `json:"enabled,omitempty"`
	FilterExpr string `json:"filter_expr,omitempty"`
}

// SubscribeEventSource creates a subscription for an event source.
func (c *Client) SubscribeEventSource(ctx context.Context, sourceID string, req SubscribeEventSourceRequest) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/event-sources", sourceID, "subscribe")
	if err != nil {
		return nil, fmt.Errorf("invalid event source id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteEventSourceSubscription removes a subscription from an event source.
func (c *Client) DeleteEventSourceSubscription(ctx context.Context, sourceID, subID string) error {
	endpoint, err := joinPath("/v1/event-sources", sourceID, "subscriptions", subID)
	if err != nil {
		return fmt.Errorf("invalid event source or subscription id: %w", err)
	}
	return c.doJSON(ctx, http.MethodDelete, endpoint, nil, nil, nil)
}

// GetServerSecret returns a single server secret by ID.
func (c *Client) GetServerSecret(ctx context.Context, secretID string) (json.RawMessage, error) {
	endpoint, err := joinPath("/v1/secrets", secretID)
	if err != nil {
		return nil, fmt.Errorf("invalid secret id: %w", err)
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListNotificationDeliveries returns notification delivery records.
func (c *Client) ListNotificationDeliveries(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doListJSON(ctx, "/v1/notification-deliveries", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListExpiringAPIKeys returns API keys that are expiring soon.
func (c *Client) ListExpiringAPIKeys(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doListJSON(ctx, "/v1/api-keys/expiring-soon", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
