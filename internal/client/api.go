package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

// ListJobs returns all jobs for the given project.
func (c *Client) ListJobs(ctx context.Context, projectID string) ([]types.Job, error) {
	query := url.Values{}
	query.Set("project_id", projectID)

	var out []types.Job
	if err := c.doListJSON(ctx, "/v1/jobs", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetJob returns a job by ID.
func (c *Client) GetJob(ctx context.Context, id string) (*types.Job, error) {
	var out types.Job
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/jobs", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateJob creates a new job.
func (c *Client) CreateJob(ctx context.Context, req CreateJobRequest) (*types.Job, error) {
	var out types.Job
	if err := c.doJSON(ctx, http.MethodPost, "/v1/jobs", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteJob deletes a job by ID.
func (c *Client) DeleteJob(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, path.Join("/v1/jobs", id), nil, nil, nil)
}

// UpdateJob updates a job by ID.
func (c *Client) UpdateJob(ctx context.Context, id string, req UpdateJobRequest) (*types.Job, error) {
	var out types.Job
	if err := c.doJSON(ctx, http.MethodPatch, path.Join("/v1/jobs", id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TriggerJob triggers a job run.
func (c *Client) TriggerJob(ctx context.Context, jobID string, req TriggerJobRequest, idempotencyKey string) (*TriggerJobResponse, error) {
	var out TriggerJobResponse
	headers := map[string]string{}
	if strings.TrimSpace(idempotencyKey) != "" {
		headers["X-Idempotency-Key"] = strings.TrimSpace(idempotencyKey)
	}
	if err := c.doJSONWithHeaders(ctx, http.MethodPost, path.Join("/v1/jobs", jobID, "trigger"), nil, req, headers, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// BulkTriggerJob triggers multiple job runs at once.
func (c *Client) BulkTriggerJob(ctx context.Context, jobID string, req BulkTriggerRequest) (*BulkTriggerResponse, error) {
	var out BulkTriggerResponse
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/jobs", jobID, "trigger", "bulk"), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListJobVersions returns all versions for a job.
func (c *Client) ListJobVersions(ctx context.Context, jobID string) ([]types.JobVersion, error) {
	var out []types.JobVersion
	if err := c.doListJSON(ctx, path.Join("/v1/jobs", jobID, "versions"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListRuns returns runs for a project, optionally filtered by status.
func (c *Client) ListRuns(ctx context.Context, projectID, status string, limit int, cursor *time.Time) ([]types.JobRun, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	if strings.TrimSpace(status) != "" {
		query.Set("status", strings.TrimSpace(status))
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if cursor != nil {
		query.Set("cursor", cursor.Format(time.RFC3339Nano))
	}

	var out []types.JobRun
	if err := c.doListJSON(ctx, "/v1/runs", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListAllRuns fetches all runs by following cursor-based pagination.
func (c *Client) ListAllRuns(ctx context.Context, projectID, status string) ([]types.JobRun, error) {
	const pageSize = 100
	var all []types.JobRun
	var cursor *time.Time

	for {
		page, err := c.ListRuns(ctx, projectID, status, pageSize, cursor)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < pageSize {
			break
		}
		last := page[len(page)-1].CreatedAt
		cursor = &last
	}
	return all, nil
}

// GetRun returns a run by ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*types.JobRun, error) {
	var out types.JobRun
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/runs", runID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelRun cancels a run by ID.
func (c *Client) CancelRun(ctx context.Context, runID string) (*types.JobRun, error) {
	var out types.JobRun
	if err := c.doJSON(ctx, http.MethodDelete, path.Join("/v1/runs", runID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRunEvents returns events for a run.
func (c *Client) ListRunEvents(ctx context.Context, runID, level, eventType string) ([]types.RunEvent, error) {
	query := url.Values{}
	if strings.TrimSpace(level) != "" {
		query.Set("level", strings.TrimSpace(level))
	}
	if strings.TrimSpace(eventType) != "" {
		query.Set("type", strings.TrimSpace(eventType))
	}

	var out []types.RunEvent
	if err := c.doListAllJSON(ctx, path.Join("/v1/runs", runID, "events"), query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Health checks the server health endpoint.
func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	var out HealthStatus
	if err := c.doJSON(ctx, http.MethodGet, "/health", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// HealthReady checks the server readiness endpoint.
func (c *Client) HealthReady(ctx context.Context) (*HealthStatus, error) {
	var out HealthStatus
	if err := c.doJSON(ctx, http.MethodGet, "/health/ready", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListWorkflows returns all workflows for a project.
func (c *Client) ListWorkflows(ctx context.Context, projectID string) ([]types.Workflow, error) {
	query := url.Values{}
	query.Set("project_id", projectID)

	var out []types.Workflow
	if err := c.doListJSON(ctx, "/v1/workflows", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflow returns a workflow by ID.
func (c *Client) GetWorkflow(ctx context.Context, workflowID string) (*WorkflowResponse, error) {
	var out WorkflowResponse
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/workflows", workflowID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateWorkflow creates a new workflow.
func (c *Client) CreateWorkflow(ctx context.Context, req CreateWorkflowRequest) (*WorkflowResponse, error) {
	var out WorkflowResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/workflows", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateWorkflow updates a workflow by ID.
func (c *Client) UpdateWorkflow(ctx context.Context, id string, req UpdateWorkflowRequest) (*WorkflowResponse, error) {
	var out WorkflowResponse
	if err := c.doJSON(ctx, http.MethodPatch, path.Join("/v1/workflows", id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteWorkflow deletes a workflow by ID.
func (c *Client) DeleteWorkflow(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, path.Join("/v1/workflows", id), nil, nil, &map[string]string{})
}

// TriggerWorkflow triggers a workflow run.
func (c *Client) TriggerWorkflow(ctx context.Context, workflowID string, req TriggerWorkflowRequest) (*types.WorkflowRun, error) {
	var out types.WorkflowRun
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/workflows", workflowID, "trigger"), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListWorkflowRuns returns runs for a workflow.
func (c *Client) ListWorkflowRuns(ctx context.Context, workflowID string, limit, offset int) ([]types.WorkflowRun, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", offset))
	}

	var out []types.WorkflowRun
	if err := c.doListJSON(ctx, path.Join("/v1/workflows", workflowID, "runs"), query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListWorkflowRunsByProject returns workflow runs for a project.
func (c *Client) ListWorkflowRunsByProject(ctx context.Context, projectID, status string, limit int) ([]types.WorkflowRun, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	if strings.TrimSpace(status) != "" {
		query.Set("status", strings.TrimSpace(status))
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}

	var out []types.WorkflowRun
	if err := c.doListJSON(ctx, "/v1/workflow-runs", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWorkflowRun returns a workflow run by ID.
func (c *Client) GetWorkflowRun(ctx context.Context, workflowRunID string) (*types.WorkflowRun, error) {
	var out types.WorkflowRun
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/workflow-runs", workflowRunID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelWorkflowRun cancels a workflow run.
func (c *Client) CancelWorkflowRun(ctx context.Context, workflowRunID string) (*types.WorkflowRun, error) {
	var out types.WorkflowRun
	if err := c.doJSON(ctx, http.MethodDelete, path.Join("/v1/workflow-runs", workflowRunID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListWorkflowStepRuns returns step runs for a workflow run.
func (c *Client) ListWorkflowStepRuns(ctx context.Context, workflowRunID string) ([]types.WorkflowStepRun, error) {
	var out []types.WorkflowStepRun
	if err := c.doListJSON(ctx, path.Join("/v1/workflow-runs", workflowRunID, "steps"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateAPIKey creates a new API key.
func (c *Client) CreateAPIKey(ctx context.Context, req CreateAPIKeyRequest) (*APIKeyCreateResponse, error) {
	var out APIKeyCreateResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/api-keys", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAPIKeys returns all API keys for a project.
func (c *Client) ListAPIKeys(ctx context.Context, projectID string) ([]types.APIKey, error) {
	query := url.Values{}
	query.Set("project_id", projectID)

	var out []types.APIKey
	if err := c.doListJSON(ctx, "/v1/api-keys", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RevokeAPIKey revokes an API key.
func (c *Client) RevokeAPIKey(ctx context.Context, keyID string) error {
	return c.doJSON(ctx, http.MethodDelete, path.Join("/v1/api-keys", keyID), nil, nil, &map[string]string{})
}

// RotateAPIKey rotates an API key.
func (c *Client) RotateAPIKey(ctx context.Context, keyID string, req RotateAPIKeyRequest) (*RotateAPIKeyResponse, error) {
	var out RotateAPIKeyResponse
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/api-keys", keyID, "rotate"), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Stats returns queue statistics.
func (c *Client) Stats(ctx context.Context) (*QueueStats, error) {
	var out QueueStats
	if err := c.doJSON(ctx, http.MethodGet, "/v1/stats", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListEventTriggers returns event triggers for a project.
func (c *Client) ListEventTriggers(ctx context.Context, projectID, status string) ([]types.EventTrigger, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	if status != "" {
		query.Set("status", status)
	}

	var out []types.EventTrigger
	if err := c.doListJSON(ctx, "/v1/events", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetEventTrigger returns an event trigger by key.
func (c *Client) GetEventTrigger(ctx context.Context, eventKey string) (*types.EventTrigger, error) {
	var out types.EventTrigger
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/events", eventKey), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SendEvent sends an event to resolve a trigger.
func (c *Client) SendEvent(ctx context.Context, eventKey string, payload map[string]any) (*types.EventTrigger, error) {
	body := map[string]any{}
	if payload != nil {
		body["payload"] = payload
	}
	var out types.EventTrigger
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/events", eventKey, "send"), nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PurgeEventTriggers purges old event triggers.
func (c *Client) PurgeEventTriggers(ctx context.Context, olderThanDays int, dryRun bool) (int64, error) {
	body := map[string]any{
		"older_than_days": olderThanDays,
		"dry_run":         dryRun,
	}
	var out map[string]any
	if err := c.doJSON(ctx, http.MethodPost, "/v1/events/purge", nil, body, &out); err != nil {
		return 0, err
	}
	if dryRun {
		if v, ok := out["would_delete"].(float64); ok {
			return int64(v), nil
		}
		return 0, nil
	}
	if v, ok := out["deleted"].(float64); ok {
		return int64(v), nil
	}
	return 0, nil
}

// ListEnvironments returns environments for a project.
func (c *Client) ListEnvironments(ctx context.Context, projectID string) ([]types.Environment, error) {
	query := url.Values{}
	query.Set("project_id", projectID)

	var out []types.Environment
	if err := c.doListJSON(ctx, "/v1/environments", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateDeploymentVersion creates a new deployment version.
func (c *Client) CreateDeploymentVersion(ctx context.Context, req CreateDeploymentVersionRequest) (*DeploymentVersion, error) {
	var out DeploymentVersion
	if err := c.doJSON(ctx, http.MethodPost, "/v1/deployments", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// FinalizeDeployment finalizes a deployment version.
func (c *Client) FinalizeDeployment(ctx context.Context, id string, req FinalizeDeploymentRequest) error {
	return c.doJSON(ctx, http.MethodPost, path.Join("/v1/deployments", id, "finalize"), nil, req, nil)
}

// PromoteDeployment promotes a deployment version.
func (c *Client) PromoteDeployment(ctx context.Context, id string, req PromoteDeploymentRequest) error {
	return c.doJSON(ctx, http.MethodPost, path.Join("/v1/deployments", id, "promote"), nil, req, nil)
}

// RollbackDeployment rolls back a deployment.
func (c *Client) RollbackDeployment(ctx context.Context, id string, req RollbackDeploymentRequest) error {
	return c.doJSON(ctx, http.MethodPost, path.Join("/v1/deployments", id, "rollback"), nil, req, nil)
}

// ListDeployments returns deployments for a project.
func (c *Client) ListDeployments(ctx context.Context, projectID string, limit int) ([]DeploymentVersion, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}

	var out []DeploymentVersion
	if err := c.doListJSON(ctx, "/v1/deployments", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListServerSecrets returns server-side secrets for a project.
func (c *Client) ListServerSecrets(ctx context.Context, projectID, environment string) ([]ServerSecret, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	if strings.TrimSpace(environment) != "" {
		query.Set("environment", strings.TrimSpace(environment))
	}

	var out []ServerSecret
	if err := c.doListJSON(ctx, "/v1/secrets", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateServerSecret creates a new server-side secret.
func (c *Client) CreateServerSecret(ctx context.Context, req CreateServerSecretRequest) (*ServerSecret, error) {
	var out ServerSecret
	if err := c.doJSON(ctx, http.MethodPost, "/v1/secrets", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteServerSecret deletes a server-side secret.
func (c *Client) DeleteServerSecret(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, path.Join("/v1/secrets", id), nil, nil, nil)
}

// GetPerformanceAnalytics returns performance analytics for a project.
func (c *Client) GetPerformanceAnalytics(ctx context.Context, projectID string, periodHours int) (*PerformanceAnalytics, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	if periodHours > 0 {
		query.Set("period_hours", fmt.Sprintf("%d", periodHours))
	}

	var out PerformanceAnalytics
	if err := c.doJSON(ctx, http.MethodGet, "/v1/analytics/performance", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListMembers returns project members.
func (c *Client) ListMembers(ctx context.Context, projectID string) ([]ProjectMember, error) {
	query := url.Values{}
	query.Set("project_id", projectID)

	var out []ProjectMember
	if err := c.doListJSON(ctx, "/v1/members", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddMember adds a member to a project.
func (c *Client) AddMember(ctx context.Context, req AssignMemberRequest) (*ProjectMember, error) {
	var out ProjectMember
	if err := c.doJSON(ctx, http.MethodPost, "/v1/members", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RemoveMember removes a member from a project.
func (c *Client) RemoveMember(ctx context.Context, userID string) error {
	return c.doJSON(ctx, http.MethodDelete, path.Join("/v1/members", userID), nil, nil, nil)
}

// ListRoles returns project roles.
func (c *Client) ListRoles(ctx context.Context, projectID string) ([]ProjectRole, error) {
	query := url.Values{}
	query.Set("project_id", projectID)

	var out []ProjectRole
	if err := c.doListJSON(ctx, "/v1/roles", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListAuditEvents returns audit events for a project.
func (c *Client) ListAuditEvents(ctx context.Context, params ListAuditEventsParams) ([]AuditEvent, error) {
	query := url.Values{}
	query.Set("project_id", params.ProjectID)
	if strings.TrimSpace(params.ActorID) != "" {
		query.Set("actor_id", strings.TrimSpace(params.ActorID))
	}
	if strings.TrimSpace(params.ResourceType) != "" {
		query.Set("resource_type", strings.TrimSpace(params.ResourceType))
	}
	if strings.TrimSpace(params.ResourceID) != "" {
		query.Set("resource_id", strings.TrimSpace(params.ResourceID))
	}
	if params.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	if params.From != nil {
		query.Set("from", params.From.UTC().Format(time.RFC3339Nano))
	}
	if params.To != nil {
		query.Set("to", params.To.UTC().Format(time.RFC3339Nano))
	}
	if strings.TrimSpace(params.Order) != "" {
		query.Set("order", strings.TrimSpace(params.Order))
	}

	var out []AuditEvent
	if err := c.doListJSON(ctx, "/v1/audit-events", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetJobBySlug looks up a job by its slug within a project.
// It passes slug as a query parameter and auto-paginates through all pages
// so that projects with many jobs never silently miss the target.
func (c *Client) GetJobBySlug(ctx context.Context, projectID, slug string) (*types.Job, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	query.Set("slug", slug)

	var jobs []types.Job
	if err := c.doListAllJSON(ctx, "/v1/jobs", query, &jobs); err != nil {
		return nil, err
	}
	for i := range jobs {
		if jobs[i].Slug == slug {
			return &jobs[i], nil
		}
	}
	return nil, fmt.Errorf("job with slug %q not found in project", slug)
}

// CreateCodeDeployment creates a new code-first deployment and returns the
// deployment record plus a presigned PUT URL for uploading the source tarball.
func (c *Client) CreateCodeDeployment(ctx context.Context, jobID string, req CreateCodeDeploymentRequest) (*CreateCodeDeploymentResponse, error) {
	var out CreateCodeDeploymentResponse
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/jobs", jobID, "deployments"), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ConfirmCodeDeployment marks the tarball upload complete, triggering the build.
func (c *Client) ConfirmCodeDeployment(ctx context.Context, jobID, deploymentID string, req ConfirmCodeDeploymentRequest) (*CodeDeployment, error) {
	var out CodeDeployment
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/jobs", jobID, "deployments", deploymentID, "confirm"), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCodeDeployment fetches a single code deployment by ID.
func (c *Client) GetCodeDeployment(ctx context.Context, jobID, deploymentID string) (*CodeDeployment, error) {
	var out CodeDeployment
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/jobs", jobID, "deployments", deploymentID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListCodeDeployments lists code deployments for a job.
func (c *Client) ListCodeDeployments(ctx context.Context, jobID string, limit int) ([]CodeDeployment, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	var out []CodeDeployment
	if err := c.doListJSON(ctx, path.Join("/v1/jobs", jobID, "deployments"), query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RollbackCodeDeployment activates a previously ready deployment as the active
// deployment for the job, effectively rolling back to that version.
func (c *Client) RollbackCodeDeployment(ctx context.Context, jobID, deploymentID, projectID string) (*CodeDeployment, error) {
	body := ConfirmCodeDeploymentRequest{ProjectID: projectID}
	var out CodeDeployment
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/jobs", jobID, "deployments", deploymentID, "rollback"), nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ServerCapabilities represents the capabilities reported by a Strait server instance.
type ServerCapabilities struct {
	CodeDeployEnabled bool   `json:"code_deploy_enabled"`
	BuildKitAddress   string `json:"buildkit_address,omitempty"`
	RegistryHost      string `json:"registry_host,omitempty"`
}

// GetServerCapabilities returns the server's feature capabilities.
// Returns an error when the capabilities endpoint is unavailable (e.g. older servers).
func (c *Client) GetServerCapabilities(ctx context.Context) (*ServerCapabilities, error) {
	var out ServerCapabilities
	if err := c.doJSON(ctx, http.MethodGet, "/v1/system/capabilities", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
