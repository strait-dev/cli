package client

import (
	"context"
	"encoding/json"
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
func (c *Client) CreateJob(ctx context.Context, req CreateJobRequest, idempotencyKey string) (*types.Job, error) {
	var out types.Job
	headers := map[string]string{}
	if strings.TrimSpace(idempotencyKey) != "" {
		headers["X-Idempotency-Key"] = strings.TrimSpace(idempotencyKey)
	}
	if err := c.doJSONWithHeaders(ctx, http.MethodPost, "/v1/jobs", nil, req, headers, &out); err != nil {
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

// BulkCancelRuns cancels many runs in a single request.
func (c *Client) BulkCancelRuns(ctx context.Context, ids []string) (*BulkCancelRunsResponse, error) {
	req := BulkCancelRunsRequest{IDs: ids}
	var out BulkCancelRunsResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/runs/bulk-cancel", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ReplayRun replays a run by ID, preserving lineage to the original run.
func (c *Client) ReplayRun(ctx context.Context, runID string) (*types.JobRun, error) {
	var out types.JobRun
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/runs", runID, "replay"), nil, nil, &out); err != nil {
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
func (c *Client) CreateWorkflow(ctx context.Context, req CreateWorkflowRequest, idempotencyKey string) (*WorkflowResponse, error) {
	var out WorkflowResponse
	headers := map[string]string{}
	if strings.TrimSpace(idempotencyKey) != "" {
		headers["X-Idempotency-Key"] = strings.TrimSpace(idempotencyKey)
	}
	if err := c.doJSONWithHeaders(ctx, http.MethodPost, "/v1/workflows", nil, req, headers, &out); err != nil {
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

// VerifyAuditChain calls GET /v1/audit-events/verify and returns the server's
// integrity report for the project's audit event HMAC chain.
func (c *Client) VerifyAuditChain(ctx context.Context, params VerifyAuditChainParams) (*AuditChainVerification, error) {
	query := url.Values{}
	if strings.TrimSpace(params.ProjectID) != "" {
		query.Set("project_id", strings.TrimSpace(params.ProjectID))
	}
	if params.Since != nil {
		query.Set("since", params.Since.UTC().Format(time.RFC3339Nano))
	}

	var out AuditChainVerification
	if err := c.doJSON(ctx, http.MethodGet, "/v1/audit-events/verify", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
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

// GetEnvironment returns an environment by ID.
func (c *Client) GetEnvironment(ctx context.Context, id string) (*types.Environment, error) {
	var out types.Environment
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/environments", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateEnvironment creates a new environment.
func (c *Client) CreateEnvironment(ctx context.Context, req CreateEnvironmentRequest) (*types.Environment, error) {
	var out types.Environment
	if err := c.doJSON(ctx, http.MethodPost, "/v1/environments", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateEnvironment updates an environment by ID.
func (c *Client) UpdateEnvironment(ctx context.Context, id string, req UpdateEnvironmentRequest) (*types.Environment, error) {
	var out types.Environment
	if err := c.doJSON(ctx, http.MethodPatch, path.Join("/v1/environments", id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteEnvironment deletes an environment by ID.
func (c *Client) DeleteEnvironment(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, path.Join("/v1/environments", id), nil, nil, &map[string]string{})
}

// ListEnvironmentVariables returns the variables map for an environment.
func (c *Client) ListEnvironmentVariables(ctx context.Context, id string) (map[string]string, error) {
	var out map[string]string
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/environments", id, "variables"), nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListWebhooks returns webhook subscriptions for a project.
func (c *Client) ListWebhooks(ctx context.Context, projectID string) ([]types.Webhook, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	var out []types.Webhook
	if err := c.doListJSON(ctx, "/v1/webhooks", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWebhook returns a webhook subscription by ID.
func (c *Client) GetWebhook(ctx context.Context, id string) (*types.Webhook, error) {
	var out types.Webhook
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/webhooks", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateWebhook creates a new webhook subscription.
func (c *Client) CreateWebhook(ctx context.Context, req CreateWebhookRequest) (*types.Webhook, error) {
	var out types.Webhook
	if err := c.doJSON(ctx, http.MethodPost, "/v1/webhooks", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateWebhook updates a webhook subscription.
func (c *Client) UpdateWebhook(ctx context.Context, id string, req UpdateWebhookRequest) (*types.Webhook, error) {
	var out types.Webhook
	if err := c.doJSON(ctx, http.MethodPatch, path.Join("/v1/webhooks", id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteWebhook deletes a webhook subscription by ID.
func (c *Client) DeleteWebhook(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, path.Join("/v1/webhooks", id), nil, nil, &map[string]string{})
}

// ListWebhookDeliveries returns delivery records for a webhook.
func (c *Client) ListWebhookDeliveries(ctx context.Context, id string, limit int) ([]types.WebhookDelivery, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	var out []types.WebhookDelivery
	if err := c.doListJSON(ctx, path.Join("/v1/webhooks", id, "deliveries"), query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RetryWebhookDelivery re-attempts a previous webhook delivery.
func (c *Client) RetryWebhookDelivery(ctx context.Context, deliveryID string) (*types.WebhookDelivery, error) {
	var out types.WebhookDelivery
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/webhook-deliveries", deliveryID, "retry"), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TestWebhook sends a synthetic test event to a webhook.
func (c *Client) TestWebhook(ctx context.Context, id string) (*TestWebhookResponse, error) {
	var out TestWebhookResponse
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/webhooks", id, "test"), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListEventSources returns event sources for a project.
func (c *Client) ListEventSources(ctx context.Context, projectID string) ([]types.EventSource, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	var out []types.EventSource
	if err := c.doListJSON(ctx, "/v1/event-sources", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetEventSource returns an event source by ID.
func (c *Client) GetEventSource(ctx context.Context, id string) (*types.EventSource, error) {
	var out types.EventSource
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/event-sources", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateEventSource creates a new event source.
func (c *Client) CreateEventSource(ctx context.Context, req CreateEventSourceRequest) (*types.EventSource, error) {
	var out types.EventSource
	if err := c.doJSON(ctx, http.MethodPost, "/v1/event-sources", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateEventSource updates an event source by ID.
func (c *Client) UpdateEventSource(ctx context.Context, id string, req UpdateEventSourceRequest) (*types.EventSource, error) {
	var out types.EventSource
	if err := c.doJSON(ctx, http.MethodPatch, path.Join("/v1/event-sources", id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteEventSource deletes an event source by ID.
func (c *Client) DeleteEventSource(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, path.Join("/v1/event-sources", id), nil, nil, &map[string]string{})
}

// ListJobGroups returns job groups for a project.
func (c *Client) ListJobGroups(ctx context.Context, projectID string) ([]types.JobGroup, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	var out []types.JobGroup
	if err := c.doListJSON(ctx, "/v1/job-groups", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetJobGroup returns a job group by ID.
func (c *Client) GetJobGroup(ctx context.Context, id string) (*types.JobGroup, error) {
	var out types.JobGroup
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/job-groups", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateJobGroup creates a new job group.
func (c *Client) CreateJobGroup(ctx context.Context, req CreateJobGroupRequest) (*types.JobGroup, error) {
	var out types.JobGroup
	if err := c.doJSON(ctx, http.MethodPost, "/v1/job-groups", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateJobGroup updates a job group by ID.
func (c *Client) UpdateJobGroup(ctx context.Context, id string, req UpdateJobGroupRequest) (*types.JobGroup, error) {
	var out types.JobGroup
	if err := c.doJSON(ctx, http.MethodPatch, path.Join("/v1/job-groups", id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteJobGroup deletes a job group by ID.
func (c *Client) DeleteJobGroup(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, path.Join("/v1/job-groups", id), nil, nil, &map[string]string{})
}

// ListJobsInGroup returns jobs that belong to a job group.
func (c *Client) ListJobsInGroup(ctx context.Context, id string) ([]types.Job, error) {
	var out []types.Job
	if err := c.doListJSON(ctx, path.Join("/v1/job-groups", id, "jobs"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PauseJobGroup pauses execution for all jobs in a group.
func (c *Client) PauseJobGroup(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodPost, path.Join("/v1/job-groups", id, "pause"), nil, nil, &map[string]string{})
}

// ResumeJobGroup resumes execution for all jobs in a group.
func (c *Client) ResumeJobGroup(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodPost, path.Join("/v1/job-groups", id, "resume"), nil, nil, &map[string]string{})
}

// GetJobGroupStats returns aggregate stats for a job group.
func (c *Client) GetJobGroupStats(ctx context.Context, id string) (*types.JobGroupStats, error) {
	var out types.JobGroupStats
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/job-groups", id, "stats"), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListNotificationChannels returns notification channels for a project.
func (c *Client) ListNotificationChannels(ctx context.Context, projectID string) ([]types.NotificationChannel, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	var out []types.NotificationChannel
	if err := c.doListJSON(ctx, "/v1/notification-channels", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetNotificationChannel returns a notification channel by ID.
func (c *Client) GetNotificationChannel(ctx context.Context, id string) (*types.NotificationChannel, error) {
	var out types.NotificationChannel
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/notification-channels", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateNotificationChannel creates a new notification channel.
func (c *Client) CreateNotificationChannel(ctx context.Context, req CreateNotificationChannelRequest) (*types.NotificationChannel, error) {
	var out types.NotificationChannel
	if err := c.doJSON(ctx, http.MethodPost, "/v1/notification-channels", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateNotificationChannel updates a notification channel by ID.
func (c *Client) UpdateNotificationChannel(ctx context.Context, id string, req UpdateNotificationChannelRequest) (*types.NotificationChannel, error) {
	var out types.NotificationChannel
	if err := c.doJSON(ctx, http.MethodPatch, path.Join("/v1/notification-channels", id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteNotificationChannel deletes a notification channel by ID.
func (c *Client) DeleteNotificationChannel(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, path.Join("/v1/notification-channels", id), nil, nil, &map[string]string{})
}

// ListLogDrains returns log drains for a project.
func (c *Client) ListLogDrains(ctx context.Context, projectID string) ([]types.LogDrain, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	var out []types.LogDrain
	if err := c.doListJSON(ctx, "/v1/log-drains", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetLogDrain returns a log drain by ID.
func (c *Client) GetLogDrain(ctx context.Context, id string) (*types.LogDrain, error) {
	var out types.LogDrain
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/log-drains", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateLogDrain creates a new log drain.
func (c *Client) CreateLogDrain(ctx context.Context, req CreateLogDrainRequest) (*types.LogDrain, error) {
	var out types.LogDrain
	if err := c.doJSON(ctx, http.MethodPost, "/v1/log-drains", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateLogDrain updates a log drain by ID.
func (c *Client) UpdateLogDrain(ctx context.Context, id string, req UpdateLogDrainRequest) (*types.LogDrain, error) {
	var out types.LogDrain
	if err := c.doJSON(ctx, http.MethodPatch, path.Join("/v1/log-drains", id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteLogDrain deletes a log drain by ID.
func (c *Client) DeleteLogDrain(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, path.Join("/v1/log-drains", id), nil, nil, &map[string]string{})
}

// CloneJob clones a job and returns the new copy.
func (c *Client) CloneJob(ctx context.Context, id string, req CloneJobRequest) (*types.Job, error) {
	var out types.Job
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/jobs", id, "clone"), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetJobHealth returns the health summary for a job.
func (c *Client) GetJobHealth(ctx context.Context, id string) (*types.JobHealth, error) {
	var out types.JobHealth
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/jobs", id, "health"), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListJobDependencies returns the dependency edges for a job.
func (c *Client) ListJobDependencies(ctx context.Context, id string) ([]types.JobDependency, error) {
	var out []types.JobDependency
	if err := c.doListJSON(ctx, path.Join("/v1/jobs", id, "dependencies"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddJobDependency creates a new dependency edge for a job.
func (c *Client) AddJobDependency(ctx context.Context, id string, req AddJobDependencyRequest) (*types.JobDependency, error) {
	var out types.JobDependency
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/jobs", id, "dependencies"), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// BatchUpdateJobs applies multiple job updates in one call.
func (c *Client) BatchUpdateJobs(ctx context.Context, req BatchUpdateJobsRequest) (*BatchUpdateJobsResponse, error) {
	var out BatchUpdateJobsResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/jobs/batch", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CloneWorkflow clones a workflow and returns the new copy.
func (c *Client) CloneWorkflow(ctx context.Context, id string, req CloneWorkflowRequest) (*WorkflowResponse, error) {
	var out WorkflowResponse
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/workflows", id, "clone"), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DryRunWorkflow performs a workflow dry-run without persisting state.
func (c *Client) DryRunWorkflow(ctx context.Context, id string, payload json.RawMessage) (json.RawMessage, error) {
	body := map[string]json.RawMessage{}
	if len(payload) > 0 {
		body["payload"] = payload
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/workflows", id, "dry-run"), nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PlanWorkflow returns the planned execution graph without running.
func (c *Client) PlanWorkflow(ctx context.Context, id string, payload json.RawMessage) (json.RawMessage, error) {
	body := map[string]json.RawMessage{}
	if len(payload) > 0 {
		body["payload"] = payload
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/workflows", id, "plan"), nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SimulateWorkflow simulates running a workflow.
func (c *Client) SimulateWorkflow(ctx context.Context, id string, payload json.RawMessage) (json.RawMessage, error) {
	body := map[string]json.RawMessage{}
	if len(payload) > 0 {
		body["payload"] = payload
	}
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/workflows", id, "simulate"), nil, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListWorkflowVersions returns the version history for a workflow.
func (c *Client) ListWorkflowVersions(ctx context.Context, id string) ([]types.WorkflowVersion, error) {
	var out []types.WorkflowVersion
	if err := c.doListJSON(ctx, path.Join("/v1/workflows", id, "versions"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DiffWorkflowVersions returns the diff between two workflow versions.
func (c *Client) DiffWorkflowVersions(ctx context.Context, id string, fromV, toV int) (*types.WorkflowDiff, error) {
	query := url.Values{}
	query.Set("from", fmt.Sprintf("%d", fromV))
	query.Set("to", fmt.Sprintf("%d", toV))
	var out types.WorkflowDiff
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/workflows", id, "diff"), query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetWorkflowPolicy returns the current run-time policy for a workflow.
func (c *Client) GetWorkflowPolicy(ctx context.Context, id string) (*types.WorkflowPolicy, error) {
	var out types.WorkflowPolicy
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/workflows", id, "policy"), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetWorkflowPolicy replaces the run-time policy for a workflow.
func (c *Client) SetWorkflowPolicy(ctx context.Context, id string, policy json.RawMessage) (*types.WorkflowPolicy, error) {
	req := SetWorkflowPolicyRequest{Policy: policy}
	var out types.WorkflowPolicy
	if err := c.doJSON(ctx, http.MethodPut, path.Join("/v1/workflows", id, "policy"), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PauseWorkflowRun pauses an in-flight workflow run.
func (c *Client) PauseWorkflowRun(ctx context.Context, id string) (*types.WorkflowRun, error) {
	var out types.WorkflowRun
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/workflow-runs", id, "pause"), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ResumeWorkflowRun resumes a paused workflow run.
func (c *Client) ResumeWorkflowRun(ctx context.Context, id string) (*types.WorkflowRun, error) {
	var out types.WorkflowRun
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/workflow-runs", id, "resume"), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RetryWorkflowRun retries a failed workflow run.
func (c *Client) RetryWorkflowRun(ctx context.Context, id string) (*types.WorkflowRun, error) {
	var out types.WorkflowRun
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/workflow-runs", id, "retry"), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ApproveWorkflowStep approves a workflow step pending review.
func (c *Client) ApproveWorkflowStep(ctx context.Context, runID, stepRef string) error {
	return c.doJSON(ctx, http.MethodPost, path.Join("/v1/workflow-runs", runID, "steps", stepRef, "approve"), nil, nil, &map[string]string{})
}

// RetryWorkflowStep retries an individual workflow step.
func (c *Client) RetryWorkflowStep(ctx context.Context, runID, stepRef string) error {
	return c.doJSON(ctx, http.MethodPost, path.Join("/v1/workflow-runs", runID, "steps", stepRef, "retry"), nil, nil, &map[string]string{})
}

// SkipWorkflowStep skips a workflow step.
func (c *Client) SkipWorkflowStep(ctx context.Context, runID, stepRef string) error {
	return c.doJSON(ctx, http.MethodPost, path.Join("/v1/workflow-runs", runID, "steps", stepRef, "skip"), nil, nil, &map[string]string{})
}

// ForceCompleteWorkflowStep marks a workflow step as completed regardless of state.
func (c *Client) ForceCompleteWorkflowStep(ctx context.Context, runID, stepRef string) error {
	return c.doJSON(ctx, http.MethodPost, path.Join("/v1/workflow-runs", runID, "steps", stepRef, "force-complete"), nil, nil, &map[string]string{})
}

// RescheduleRun reschedules a run for a future execution time.
func (c *Client) RescheduleRun(ctx context.Context, runID string, at time.Time) (*types.JobRun, error) {
	req := RescheduleRunRequest{ScheduledAt: at}
	var out types.JobRun
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/runs", runID, "reschedule"), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListDLQ returns runs that have been moved to the dead letter queue.
func (c *Client) ListDLQ(ctx context.Context, projectID string) ([]types.DLQRun, error) {
	query := url.Values{}
	if projectID != "" {
		query.Set("project_id", projectID)
	}
	var out []types.DLQRun
	if err := c.doListJSON(ctx, "/v1/runs/dlq", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ReplayDLQ replays a run from the dead letter queue.
func (c *Client) ReplayDLQ(ctx context.Context, dlqID string) (*types.JobRun, error) {
	var out types.JobRun
	if err := c.doJSON(ctx, http.MethodPost, path.Join("/v1/runs/dlq", dlqID, "replay"), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRunOutputs returns the structured outputs produced by a run.
func (c *Client) ListRunOutputs(ctx context.Context, runID string) ([]types.RunOutput, error) {
	var out []types.RunOutput
	if err := c.doListJSON(ctx, path.Join("/v1/runs", runID, "outputs"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListRunToolCalls returns the tool calls invoked during a run.
func (c *Client) ListRunToolCalls(ctx context.Context, runID string) ([]types.RunToolCall, error) {
	var out []types.RunToolCall
	if err := c.doListJSON(ctx, path.Join("/v1/runs", runID, "tool-calls"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetRunUsage returns resource usage for a run.
func (c *Client) GetRunUsage(ctx context.Context, runID string) (*types.RunUsage, error) {
	var out types.RunUsage
	if err := c.doJSON(ctx, http.MethodGet, path.Join("/v1/runs", runID, "usage"), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRunCheckpoints returns checkpoints recorded during a run.
func (c *Client) ListRunCheckpoints(ctx context.Context, runID string) ([]types.RunCheckpoint, error) {
	var out []types.RunCheckpoint
	if err := c.doListJSON(ctx, path.Join("/v1/runs", runID, "checkpoints"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCurrentUsage returns the active billing period usage.
func (c *Client) GetCurrentUsage(ctx context.Context) (*types.UsagePeriod, error) {
	var out types.UsagePeriod
	if err := c.doJSON(ctx, http.MethodGet, "/v1/billing/usage", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUsageHistory returns historical billing periods.
func (c *Client) GetUsageHistory(ctx context.Context) ([]types.UsagePeriod, error) {
	var out []types.UsagePeriod
	if err := c.doListJSON(ctx, "/v1/billing/usage/history", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetUsageForecast returns projected end-of-period usage.
func (c *Client) GetUsageForecast(ctx context.Context) (*types.UsagePeriod, error) {
	var out types.UsagePeriod
	if err := c.doJSON(ctx, http.MethodGet, "/v1/billing/usage/forecast", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCostsAnalytics returns cost analytics for a project.
func (c *Client) GetCostsAnalytics(ctx context.Context, projectID string, periodHours int) (*types.CostsAnalytics, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	query.Set("period_hours", fmt.Sprintf("%d", periodHours))
	var out types.CostsAnalytics
	if err := c.doJSON(ctx, http.MethodGet, "/v1/analytics/costs", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetReliabilityAnalytics returns reliability metrics for a project.
func (c *Client) GetReliabilityAnalytics(ctx context.Context, projectID string, periodHours int) (*types.ReliabilityAnalytics, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	query.Set("period_hours", fmt.Sprintf("%d", periodHours))
	var out types.ReliabilityAnalytics
	if err := c.doJSON(ctx, http.MethodGet, "/v1/analytics/reliability", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListTopFailingJobs returns the jobs with the highest failure rate.
func (c *Client) ListTopFailingJobs(ctx context.Context, projectID string, periodHours, limit int) ([]types.TopFailingJob, error) {
	query := url.Values{}
	query.Set("project_id", projectID)
	query.Set("period_hours", fmt.Sprintf("%d", periodHours))
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	var out []types.TopFailingJob
	if err := c.doListJSON(ctx, "/v1/analytics/top-failing", query, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListTeamPolicies returns the RBAC policies attached to the team.
func (c *Client) ListTeamPolicies(ctx context.Context) ([]types.TeamPolicy, error) {
	var out []types.TeamPolicy
	if err := c.doListJSON(ctx, "/v1/team/policies", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateTeamPolicy creates a new RBAC policy on the team.
func (c *Client) CreateTeamPolicy(ctx context.Context, req CreateTeamPolicyRequest) (*types.TeamPolicy, error) {
	var out types.TeamPolicy
	if err := c.doJSON(ctx, http.MethodPost, "/v1/team/policies", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteTeamPolicy removes an RBAC policy from the team.
func (c *Client) DeleteTeamPolicy(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, path.Join("/v1/team/policies", id), nil, nil, &map[string]string{})
}
