package client

import (
	"encoding/json"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

// CreateJobRequest is the request body for creating a job.
type CreateJobRequest struct {
	ProjectID   string          `json:"project_id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description string          `json:"description,omitempty"`
	Cron        string          `json:"cron,omitempty"`
	EndpointURL string          `json:"endpoint_url"`
	MaxAttempts int             `json:"max_attempts,omitempty"`
	TimeoutSecs int             `json:"timeout_secs,omitempty"`
	RunTTLSecs  int             `json:"run_ttl_secs,omitempty"`
	Schema      json.RawMessage `json:"payload_schema,omitempty"`
}

// UpdateJobRequest is the request body for updating a job.
type UpdateJobRequest struct {
	Name          *string          `json:"name,omitempty"`
	Slug          *string          `json:"slug,omitempty"`
	Description   *string          `json:"description,omitempty"`
	Cron          *string          `json:"cron,omitempty"`
	EndpointURL   *string          `json:"endpoint_url,omitempty"`
	MaxAttempts   *int             `json:"max_attempts,omitempty"`
	TimeoutSecs   *int             `json:"timeout_secs,omitempty"`
	RunTTLSecs    *int             `json:"run_ttl_secs,omitempty"`
	Enabled       *bool            `json:"enabled,omitempty"`
	Schema        *json.RawMessage `json:"payload_schema,omitempty"`
	ImageURI      *string          `json:"image_uri,omitempty"`
	MachinePreset *string          `json:"machine_preset,omitempty"`
	Region        *string          `json:"region,omitempty"`
}

// TriggerJobRequest is the request body for triggering a job.
type TriggerJobRequest struct {
	Payload     json.RawMessage `json:"payload,omitempty"`
	ScheduledAt *time.Time      `json:"scheduled_at,omitempty"`
	Priority    int             `json:"priority,omitempty"`
}

// TriggerJobResponse is the response from triggering a job.
type TriggerJobResponse struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	RunToken       string `json:"run_token,omitempty"`
	PayloadHash    string `json:"payload_hash,omitempty"`
	IdempotencyHit bool   `json:"idempotency_hit"`
}

// BulkTriggerItem represents a single item in a bulk trigger request.
type BulkTriggerItem struct {
	Payload        json.RawMessage `json:"payload,omitempty"`
	ScheduledAt    *time.Time      `json:"scheduled_at,omitempty"`
	Priority       int             `json:"priority,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
}

// BulkTriggerRequest is the request body for bulk triggering a job.
type BulkTriggerRequest struct {
	Items []BulkTriggerItem `json:"items"`
}

// BulkTriggerResult represents a single result in a bulk trigger response.
type BulkTriggerResult struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	RunToken       string `json:"run_token,omitempty"`
	IdempotencyHit bool   `json:"idempotency_hit"`
}

// BulkTriggerResponse is the response from bulk triggering a job.
type BulkTriggerResponse struct {
	Results []BulkTriggerResult `json:"results"`
	Total   int                 `json:"total"`
	Created int                 `json:"created"`
}

// HealthStatus represents the server health status.
type HealthStatus struct {
	Status string `json:"status"`
}

// WorkflowStepRequest is a step definition in workflow create/update requests.
type WorkflowStepRequest struct {
	JobID     string          `json:"job_id"`
	StepRef   string          `json:"step_ref"`
	DependsOn []string        `json:"depends_on,omitempty"`
	Condition json.RawMessage `json:"condition,omitempty"`
	OnFailure string          `json:"on_failure,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// CreateWorkflowRequest is the request body for creating a workflow.
type CreateWorkflowRequest struct {
	ProjectID   string                `json:"project_id"`
	Name        string                `json:"name"`
	Slug        string                `json:"slug"`
	Description string                `json:"description,omitempty"`
	Enabled     *bool                 `json:"enabled,omitempty"`
	Steps       []WorkflowStepRequest `json:"steps,omitempty"`
}

// UpdateWorkflowRequest is the request body for updating a workflow.
type UpdateWorkflowRequest struct {
	Name        *string                `json:"name,omitempty"`
	Slug        *string                `json:"slug,omitempty"`
	Description *string                `json:"description,omitempty"`
	Enabled     *bool                  `json:"enabled,omitempty"`
	Steps       *[]WorkflowStepRequest `json:"steps,omitempty"`
}

// WorkflowResponse is the response for workflow endpoints.
type WorkflowResponse struct {
	types.Workflow
	Steps []types.WorkflowStep `json:"steps"`
}

// TriggerWorkflowRequest is the request body for triggering a workflow.
type TriggerWorkflowRequest struct {
	ProjectID   string          `json:"project_id,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	TriggeredBy string          `json:"triggered_by,omitempty"`
}

// CreateAPIKeyRequest is the request body for creating an API key.
type CreateAPIKeyRequest struct {
	ProjectID string   `json:"project_id"`
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes,omitempty"`
}

// APIKeyCreateResponse is the response from creating an API key.
type APIKeyCreateResponse struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	Name      string     `json:"name"`
	Key       string     `json:"key"`
	KeyPrefix string     `json:"key_prefix"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// RotateAPIKeyRequest is the request body for rotating an API key.
type RotateAPIKeyRequest struct {
	GracePeriodMinutes int `json:"grace_period_minutes,omitempty"`
}

// RotateAPIKeyResponse is the response from rotating an API key.
type RotateAPIKeyResponse struct {
	OldKeyID       string     `json:"old_key_id"`
	NewKeyID       string     `json:"new_key_id"`
	ProjectID      string     `json:"project_id"`
	Name           string     `json:"name"`
	Key            string     `json:"key"`
	KeyPrefix      string     `json:"key_prefix"`
	Scopes         []string   `json:"scopes"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	GraceExpiresAt time.Time  `json:"grace_expires_at"`
}

// QueueStats represents queue statistics.
type QueueStats struct {
	Queued    int `json:"queued"`
	Executing int `json:"executing"`
	Delayed   int `json:"delayed"`
}

// CreateDeploymentVersionRequest is the request body for creating a deployment.
type CreateDeploymentVersionRequest struct {
	ProjectID      string `json:"project_id"`
	Environment    string `json:"environment"`
	Runtime        string `json:"runtime"`
	Manifest       any    `json:"manifest,omitempty"`
	Checksum       string `json:"checksum,omitempty"`
	ArtifactURI    string `json:"artifact_uri"`
	Strategy       string `json:"strategy,omitempty"`
	CanaryPercent  int    `json:"canary_percent,omitempty"`
	CanaryDuration string `json:"canary_duration,omitempty"`
}

// DeploymentVersion represents a deployment version.
type DeploymentVersion struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Environment string    `json:"environment"`
	Status      string    `json:"status"`
	Checksum    string    `json:"checksum,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// FinalizeDeploymentRequest is the request body for finalizing a deployment.
type FinalizeDeploymentRequest struct {
	ProjectID   string `json:"project_id"`
	Environment string `json:"environment"`
}

// PromoteDeploymentRequest is the request body for promoting a deployment.
type PromoteDeploymentRequest struct {
	ProjectID   string `json:"project_id"`
	Environment string `json:"environment"`
}

// RollbackDeploymentRequest is the request body for rolling back a deployment.
type RollbackDeploymentRequest struct {
	ProjectID   string `json:"project_id"`
	Environment string `json:"environment"`
}

// ServerSecret represents a server-side secret.
type ServerSecret struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	SecretKey   string    `json:"secret_key"`
	Environment string    `json:"environment"`
	JobID       string    `json:"job_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateServerSecretRequest is the request body for creating a secret.
type CreateServerSecretRequest struct {
	ProjectID   string `json:"project_id"`
	SecretKey   string `json:"secret_key"`
	SecretValue string `json:"secret_value"`
	Environment string `json:"environment"`
	JobID       string `json:"job_id,omitempty"`
}

// PerformanceAnalytics contains performance analytics data.
type PerformanceAnalytics struct {
	SlowestJobs   []JobPerformance `json:"slowest_jobs"`
	Throughput    ThroughputStats  `json:"throughput"`
	HealthSummary HealthSummary    `json:"health_summary"`
}

// JobPerformance contains performance data for a job.
type JobPerformance struct {
	JobID           string  `json:"job_id"`
	JobSlug         string  `json:"job_slug"`
	AvgDurationSecs float64 `json:"avg_duration_secs"`
	P95DurationSecs float64 `json:"p95_duration_secs"`
	TotalRuns       int     `json:"total_runs"`
	FailedRuns      int     `json:"failed_runs"`
}

// ThroughputStats contains throughput statistics.
type ThroughputStats struct {
	Completed   int `json:"completed"`
	Failed      int `json:"failed"`
	TimedOut    int `json:"timed_out"`
	Canceled    int `json:"canceled"`
	PeriodHours int `json:"period_hours"`
}

// HealthSummary contains health summary data.
type HealthSummary struct {
	TotalJobs       int     `json:"total_jobs"`
	ActiveJobs      int     `json:"active_jobs"`
	SuccessRate     float64 `json:"success_rate"`
	AvgDurationSecs float64 `json:"avg_duration_secs"`
	QueueDepth      int     `json:"queue_depth"`
}

// ProjectMember is an alias for the project member role type.
type ProjectMember = types.ProjectMemberRole

// AssignMemberRequest is the request body for assigning a member.
type AssignMemberRequest struct {
	UserID string `json:"user_id"`
	RoleID string `json:"role_id"`
}

// ProjectRole is an alias for the project role type.
type ProjectRole = types.ProjectRole

// AuditEvent is an alias for the audit event type.
type AuditEvent = types.AuditEvent

// ListAuditEventsParams contains parameters for listing audit events.
type ListAuditEventsParams struct {
	ProjectID    string
	ActorID      string
	ResourceType string
	ResourceID   string
	Limit        int
	From         *time.Time
	To           *time.Time
	Order        string
}

// CodeDeployment represents a code-first job deployment as returned by the API.
type CodeDeployment struct {
	ID               string     `json:"id"`
	JobID            string     `json:"job_id"`
	ProjectID        string     `json:"project_id"`
	Version          int        `json:"version"`
	Status           string     `json:"status"`
	Runtime          string     `json:"runtime"`
	SourceHash       string     `json:"source_hash"`
	SourceSizeBytes  int64      `json:"source_size_bytes"`
	SourceURI        string     `json:"source_uri"`
	BuiltImageURI    string     `json:"built_image_uri,omitempty"`
	BuiltImageDigest string     `json:"built_image_digest,omitempty"`
	BuildLogs        string     `json:"build_logs,omitempty"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	CreatedBy        string     `json:"created_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
}

// CreateCodeDeploymentRequest is the request body for creating a code deployment.
type CreateCodeDeploymentRequest struct {
	ProjectID       string `json:"project_id"`
	JobID           string `json:"job_id"`
	Runtime         string `json:"runtime"`
	SourceHash      string `json:"source_hash"`
	SourceSizeBytes int64  `json:"source_size_bytes"`
}

// CreateCodeDeploymentResponse is returned when creating a code deployment.
type CreateCodeDeploymentResponse struct {
	Deployment *CodeDeployment `json:"deployment"`
	UploadURL  string          `json:"upload_url"`
}

// ConfirmCodeDeploymentRequest is the request body for confirming a code deployment.
type ConfirmCodeDeploymentRequest struct {
	ProjectID string `json:"project_id"`
}

// DeploymentLogChunk is a single chunk published on the build log SSE stream.
type DeploymentLogChunk struct {
	Chunk string `json:"chunk,omitempty"`
	Done  bool   `json:"done,omitempty"`
}

// DeviceCodeResponse is returned by the device authorization endpoint.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// DeviceTokenResponse is returned when the device code has been approved.
type DeviceTokenResponse struct {
	APIKey    string   `json:"api_key"`
	ProjectID string   `json:"project_id"`
	Scopes    []string `json:"scopes"`
}

// RunStreamMessage represents a message from the run event stream.
type RunStreamMessage struct {
	Type      string          `json:"type"`
	EventType string          `json:"event_type,omitempty"`
	RunID     string          `json:"run_id,omitempty"`
	JobID     string          `json:"job_id,omitempty"`
	ProjectID string          `json:"project_id,omitempty"`
	Level     string          `json:"level,omitempty"`
	Message   string          `json:"message,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	From      string          `json:"from,omitempty"`
	To        string          `json:"to,omitempty"`
	Error     string          `json:"error,omitempty"`
}
