// Package types defines CLI-own response types that match the Strait REST API
// JSON contract. These types are decoupled from the server's internal domain
// package -- the REST API is the single source of truth.
package types

import (
	"encoding/json"
	"time"
)

// RunStatus represents the status of a job run.
type RunStatus string

const (
	StatusDelayed      RunStatus = "delayed"
	StatusQueued       RunStatus = "queued"
	StatusDequeued     RunStatus = "dequeued"
	StatusExecuting    RunStatus = "executing"
	StatusWaiting      RunStatus = "waiting"
	StatusCompleted    RunStatus = "completed"
	StatusFailed       RunStatus = "failed"
	StatusTimedOut     RunStatus = "timed_out"
	StatusCrashed      RunStatus = "crashed"
	StatusSystemFailed RunStatus = "system_failed"
	StatusCanceled     RunStatus = "canceled"
	StatusExpired      RunStatus = "expired"
	StatusDeadLetter   RunStatus = "dead_letter"
	StatusReplayStaged RunStatus = "replay_staged"
	StatusPaused       RunStatus = "paused"
)

// IsTerminal returns true if the status represents a final state.
func (s RunStatus) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusTimedOut, StatusCrashed, StatusSystemFailed, StatusCanceled, StatusExpired:
		return true
	default:
		return false
	}
}

// IsValid returns true if the status is a known value.
func (s RunStatus) IsValid() bool {
	switch s {
	case StatusDelayed, StatusQueued, StatusDequeued, StatusExecuting, StatusWaiting,
		StatusCompleted, StatusFailed, StatusTimedOut, StatusCrashed, StatusSystemFailed,
		StatusCanceled, StatusExpired, StatusDeadLetter, StatusReplayStaged, StatusPaused:
		return true
	default:
		return false
	}
}

// TerminalStatuses returns all terminal run statuses.
func TerminalStatuses() []RunStatus {
	return []RunStatus{
		StatusCompleted, StatusFailed, StatusTimedOut, StatusCrashed,
		StatusSystemFailed, StatusCanceled, StatusExpired,
	}
}

// Trigger source constants.
const (
	TriggerManual        = "manual"
	TriggerCron          = "cron"
	TriggerSpawn         = "spawn"
	TriggerWorkflow      = "workflow"
	TriggerRetry         = "retry"
	TriggerDebounce      = "debounce"
	TriggerJobCompletion = "job_completion"
)

// Webhook event constants.
const (
	WebhookEventRunCompleted         = "run.completed"
	WebhookEventRunFailed            = "run.failed"
	WebhookEventRunTimedOut          = "run.timed_out"
	WebhookEventRunCanceled          = "run.canceled"
	WebhookEventWorkflowCompleted    = "workflow.completed"
	WebhookEventWorkflowFailed       = "workflow.failed"
	WebhookEventComputeBudgetWarning = "compute_budget_warning"
	WebhookEventSLOBudgetWarning     = "slo.budget_warning"
)

// EventType represents the type of a run event.
type EventType string

const (
	EventLog         EventType = "log"
	EventStateChange EventType = "state_change"
	EventError       EventType = "error"
	EventProgress    EventType = "progress"
)

// WorkflowRunStatus represents the status of a workflow run.
type WorkflowRunStatus string

const (
	WfStatusPending   WorkflowRunStatus = "pending"
	WfStatusRunning   WorkflowRunStatus = "running"
	WfStatusPaused    WorkflowRunStatus = "paused"
	WfStatusCompleted WorkflowRunStatus = "completed"
	WfStatusFailed    WorkflowRunStatus = "failed"
	WfStatusTimedOut  WorkflowRunStatus = "timed_out"
	WfStatusCanceled  WorkflowRunStatus = "canceled"
)

// IsTerminal returns true if the workflow run status is a final state.
func (s WorkflowRunStatus) IsTerminal() bool {
	switch s {
	case WfStatusCompleted, WfStatusFailed, WfStatusTimedOut, WfStatusCanceled:
		return true
	default:
		return false
	}
}

// IsValid returns true if the workflow run status is a known value.
func (s WorkflowRunStatus) IsValid() bool {
	switch s {
	case WfStatusPending, WfStatusRunning, WfStatusPaused, WfStatusCompleted, WfStatusFailed, WfStatusTimedOut, WfStatusCanceled:
		return true
	default:
		return false
	}
}

// StepRunStatus represents the status of a workflow step run.
type StepRunStatus string

const (
	StepPending   StepRunStatus = "pending"
	StepWaiting   StepRunStatus = "waiting"
	StepRunning   StepRunStatus = "running"
	StepCompleted StepRunStatus = "completed"
	StepFailed    StepRunStatus = "failed"
	StepSkipped   StepRunStatus = "skipped"
	StepCanceled  StepRunStatus = "canceled"
)

// IsTerminal returns true if the step run status is a final state.
func (s StepRunStatus) IsTerminal() bool {
	switch s {
	case StepCompleted, StepFailed, StepSkipped, StepCanceled:
		return true
	default:
		return false
	}
}

// FailurePolicy determines what happens when a workflow step fails.
type FailurePolicy string

const (
	FailWorkflow   FailurePolicy = "fail_workflow"
	SkipDependents FailurePolicy = "skip_dependents"
	Continue       FailurePolicy = "continue"
)

// WorkflowStepType represents the type of a workflow step.
type WorkflowStepType string

const (
	WorkflowStepTypeJob          WorkflowStepType = "job"
	WorkflowStepTypeApproval     WorkflowStepType = "approval"
	WorkflowStepTypeSubWorkflow  WorkflowStepType = "sub_workflow"
	WorkflowStepTypeWaitForEvent WorkflowStepType = "wait_for_event"
	WorkflowStepTypeSleep        WorkflowStepType = "sleep"
)

// Approval status constants.
const (
	ApprovalStatusPending  = "pending"
	ApprovalStatusApproved = "approved"
	ApprovalStatusRejected = "rejected"
)

// Event trigger status constants.
const (
	EventTriggerStatusWaiting  = "waiting"
	EventTriggerStatusReceived = "received"
	EventTriggerStatusTimedOut = "timed_out"
	EventTriggerStatusCanceled = "canceled"
)

// Event trigger source types.
const (
	EventSourceWorkflowStep = "workflow_step"
	EventSourceJobRun       = "job_run"
)

// Trigger type constants.
const (
	TriggerTypeEvent = "event"
	TriggerTypeSleep = "sleep"
)

// VersionPolicy controls how queued runs handle new job/workflow deployments.
type VersionPolicy string

const (
	VersionPolicyPin    VersionPolicy = "pin"
	VersionPolicyLatest VersionPolicy = "latest"
	VersionPolicyMinor  VersionPolicy = "minor"
)

// IsValid returns true if the version policy is a known value.
func (p VersionPolicy) IsValid() bool {
	switch p {
	case VersionPolicyPin, VersionPolicyLatest, VersionPolicyMinor:
		return true
	default:
		return false
	}
}

// ExecutionMode determines how a job run is dispatched.
type ExecutionMode string

const (
	ExecutionModeHTTP ExecutionMode = "http"
)

// IsValid returns true if the execution mode is a known value.
func (m ExecutionMode) IsValid() bool {
	switch m {
	case ExecutionModeHTTP:
		return true
	default:
		return false
	}
}

// RetryBackoffPolicy defines the backoff strategy for step retries.
type RetryBackoffPolicy string

const (
	RetryBackoffExponential RetryBackoffPolicy = "exponential"
	RetryBackoffFixed       RetryBackoffPolicy = "fixed"
)

// RateLimitKey defines a named rate limit bucket within a job.
type RateLimitKey struct {
	Name       string `json:"name"`
	Max        int    `json:"max"`
	WindowSecs int    `json:"window_secs"`
}

// ExecutionTrace holds timing breakdown data for a run.
type ExecutionTrace struct {
	QueuedAt   *time.Time `json:"queued_at,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	QueueMs    int64      `json:"queue_ms,omitempty"`
	ExecMs     int64      `json:"exec_ms,omitempty"`
	TotalMs    int64      `json:"total_ms,omitempty"`
}

// Job represents a job definition as returned by the REST API.
type Job struct {
	ID                        string            `json:"id"`
	ProjectID                 string            `json:"project_id"`
	GroupID                   string            `json:"group_id,omitempty"`
	Name                      string            `json:"name"`
	Slug                      string            `json:"slug"`
	Description               string            `json:"description,omitempty"`
	Cron                      string            `json:"cron,omitempty"`
	PayloadSchema             json.RawMessage   `json:"payload_schema,omitempty"`
	Tags                      map[string]string `json:"tags,omitempty"`
	EndpointURL               string            `json:"endpoint_url"`
	FallbackEndpointURL       string            `json:"fallback_endpoint_url,omitempty"`
	MaxAttempts               int               `json:"max_attempts"`
	TimeoutSecs               int               `json:"timeout_secs"`
	MaxConcurrency            int               `json:"max_concurrency,omitempty"`
	MaxConcurrencyPerKey      int               `json:"max_concurrency_per_key,omitempty"`
	ExecutionWindowCron       string            `json:"execution_window_cron,omitempty"`
	Timezone                  string            `json:"timezone,omitempty"`
	RateLimitMax              int               `json:"rate_limit_max,omitempty"`
	RateLimitWindowSecs       int               `json:"rate_limit_window_secs,omitempty"`
	RateLimitKeys             []RateLimitKey    `json:"rate_limit_keys,omitempty"`
	DedupWindowSecs           int               `json:"dedup_window_secs,omitempty"`
	Enabled                   bool              `json:"enabled"`
	WebhookURL                string            `json:"webhook_url,omitempty"`
	WebhookSecret             string            `json:"webhook_secret,omitempty"`
	RunTTLSecs                int               `json:"run_ttl_secs,omitempty"`
	RetryStrategy             string            `json:"retry_strategy,omitempty"`
	RetryDelaysSecs           []int             `json:"retry_delays_secs,omitempty"`
	RetryPriorityBoost        int               `json:"retry_priority_boost,omitempty"`
	DLQAlertThreshold         *int              `json:"dlq_alert_threshold,omitempty"`
	QueueDepthAlertThreshold  *int              `json:"queue_depth_alert_threshold,omitempty"`
	EnvironmentID             string            `json:"environment_id,omitempty"`
	DefaultRunMetadata        map[string]string `json:"default_run_metadata,omitempty"`
	Version                   int               `json:"version"`
	VersionID                 string            `json:"version_id,omitempty"`
	VersionPolicy             VersionPolicy     `json:"version_policy,omitempty"`
	BackwardsCompatible       bool              `json:"backwards_compatible,omitempty"`
	SkipIfRunning             bool              `json:"skip_if_running,omitempty"`
	ResultSchema              json.RawMessage   `json:"result_schema,omitempty"`
	DebounceWindowSecs        int               `json:"debounce_window_secs,omitempty"`
	BatchWindowSecs           int               `json:"batch_window_secs,omitempty"`
	BatchMaxSize              int               `json:"batch_max_size,omitempty"`
	ExecutionMode             ExecutionMode     `json:"execution_mode,omitempty"`
	OnCompleteTriggerWorkflow string            `json:"on_complete_trigger_workflow,omitempty"`
	OnCompletePayloadMapping  json.RawMessage   `json:"on_complete_payload_mapping,omitempty"`
	MaxTokensPerRun           int64             `json:"max_tokens_per_run,omitempty"`
	MaxToolCallsPerRun        int               `json:"max_tool_calls_per_run,omitempty"`
	MaxIterationsPerRun       int               `json:"max_iterations_per_run,omitempty"`
	AllowedTools              []string          `json:"allowed_tools,omitempty"`
	BlockedTools              []string          `json:"blocked_tools,omitempty"`
	SourceType                string            `json:"source_type,omitempty"`
	ActiveDeploymentID        string            `json:"active_deployment_id,omitempty"`
	CreatedBy                 string            `json:"created_by,omitempty"`
	UpdatedBy                 string            `json:"updated_by,omitempty"`
	CreatedAt                 time.Time         `json:"created_at"`
	UpdatedAt                 time.Time         `json:"updated_at"`
}

// JobVersion represents a versioned snapshot of a job definition.
type JobVersion struct {
	ID                  string            `json:"id"`
	JobID               string            `json:"job_id"`
	Version             int               `json:"version"`
	VersionID           string            `json:"version_id,omitempty"`
	BackwardsCompatible bool              `json:"backwards_compatible,omitempty"`
	Name                string            `json:"name"`
	Slug                string            `json:"slug"`
	Description         string            `json:"description,omitempty"`
	Cron                string            `json:"cron,omitempty"`
	PayloadSchema       json.RawMessage   `json:"payload_schema,omitempty"`
	Tags                map[string]string `json:"tags,omitempty"`
	EndpointURL         string            `json:"endpoint_url"`
	FallbackEndpointURL string            `json:"fallback_endpoint_url,omitempty"`
	MaxAttempts         int               `json:"max_attempts"`
	TimeoutSecs         int               `json:"timeout_secs"`
	WebhookURL          string            `json:"webhook_url,omitempty"`
	WebhookSecret       string            `json:"webhook_secret,omitempty"`
	RunTTLSecs          int               `json:"run_ttl_secs,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
}

// JobRun represents a job run as returned by the REST API.
type JobRun struct {
	ID                    string            `json:"id"`
	JobID                 string            `json:"job_id"`
	ProjectID             string            `json:"project_id"`
	Tags                  map[string]string `json:"tags,omitempty"`
	Status                RunStatus         `json:"status"`
	Attempt               int               `json:"attempt"`
	Payload               json.RawMessage   `json:"payload,omitempty"`
	Result                json.RawMessage   `json:"result,omitempty"`
	Metadata              map[string]string `json:"metadata,omitempty"`
	Error                 string            `json:"error,omitempty"`
	ErrorClass            string            `json:"error_class,omitempty"`
	TriggeredBy           string            `json:"triggered_by"`
	ScheduledAt           *time.Time        `json:"scheduled_at,omitempty"`
	StartedAt             *time.Time        `json:"started_at,omitempty"`
	FinishedAt            *time.Time        `json:"finished_at,omitempty"`
	HeartbeatAt           *time.Time        `json:"heartbeat_at,omitempty"`
	NextRetryAt           *time.Time        `json:"next_retry_at,omitempty"`
	ExpiresAt             *time.Time        `json:"expires_at,omitempty"`
	ParentRunID           string            `json:"parent_run_id,omitempty"`
	Priority              int               `json:"priority"`
	IdempotencyKey        string            `json:"idempotency_key,omitempty"`
	JobVersion            int               `json:"job_version"`
	JobVersionID          string            `json:"job_version_id,omitempty"`
	WorkflowStepRunID     string            `json:"workflow_step_run_id,omitempty"`
	MaxAttemptsOverride   int               `json:"max_attempts_override,omitempty"`
	TimeoutSecsOverride   int               `json:"timeout_secs_override,omitempty"`
	RetryBackoff          string            `json:"retry_backoff,omitempty"`
	RetryInitialDelaySecs int               `json:"retry_initial_delay_secs,omitempty"`
	RetryMaxDelaySecs     int               `json:"retry_max_delay_secs,omitempty"`
	ExecutionTrace        *ExecutionTrace   `json:"execution_trace,omitempty"`
	DebugMode             bool              `json:"debug_mode"`
	ContinuationOf        string            `json:"continuation_of,omitempty"`
	LineageDepth          int               `json:"lineage_depth"`
	CreatedBy             string            `json:"created_by,omitempty"`
	BatchID               string            `json:"batch_id,omitempty"`
	ConcurrencyKey        string            `json:"concurrency_key,omitempty"`
	ExecutionMode         ExecutionMode     `json:"execution_mode,omitempty"`
	CreatedAt             time.Time         `json:"created_at"`
}

// RunEvent represents a log/state-change event for a run.
type RunEvent struct {
	ID        string          `json:"id"`
	RunID     string          `json:"run_id"`
	Type      EventType       `json:"type"`
	Level     string          `json:"level"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// Workflow represents a workflow DAG definition.
type Workflow struct {
	ID                  string            `json:"id"`
	ProjectID           string            `json:"project_id"`
	Name                string            `json:"name"`
	Slug                string            `json:"slug"`
	Description         string            `json:"description,omitempty"`
	Tags                map[string]string `json:"tags,omitempty"`
	Enabled             bool              `json:"enabled"`
	Version             int               `json:"version"`
	TimeoutSecs         int               `json:"timeout_secs,omitempty"`
	MaxConcurrentRuns   int               `json:"max_concurrent_runs,omitempty"`
	MaxParallelSteps    int               `json:"max_parallel_steps,omitempty"`
	Cron                string            `json:"cron,omitempty"`
	CronTimezone        string            `json:"cron_timezone,omitempty"`
	SkipIfRunning       bool              `json:"skip_if_running,omitempty"`
	VersionID           string            `json:"version_id,omitempty"`
	VersionPolicy       VersionPolicy     `json:"version_policy,omitempty"`
	BackwardsCompatible bool              `json:"backwards_compatible,omitempty"`
	CreatedBy           string            `json:"created_by,omitempty"`
	UpdatedBy           string            `json:"updated_by,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

// WorkflowStep represents a step (node) within a workflow DAG.
type WorkflowStep struct {
	ID                        string             `json:"id"`
	WorkflowID                string             `json:"workflow_id"`
	JobID                     string             `json:"job_id,omitempty"`
	StepRef                   string             `json:"step_ref"`
	DependsOn                 []string           `json:"depends_on"`
	Condition                 json.RawMessage    `json:"condition,omitempty"`
	OnFailure                 FailurePolicy      `json:"on_failure"`
	Payload                   json.RawMessage    `json:"payload,omitempty"`
	StepType                  WorkflowStepType   `json:"step_type,omitempty"`
	ApprovalTimeoutSecs       int                `json:"approval_timeout_secs,omitempty"`
	ApprovalApprovers         []string           `json:"approval_approvers,omitempty"`
	RetryMaxAttempts          int                `json:"retry_max_attempts,omitempty"`
	RetryBackoff              RetryBackoffPolicy `json:"retry_backoff,omitempty"`
	RetryInitialDelaySecs     int                `json:"retry_initial_delay_secs,omitempty"`
	RetryMaxDelaySecs         int                `json:"retry_max_delay_secs,omitempty"`
	TimeoutSecsOverride       int                `json:"timeout_secs_override,omitempty"`
	OutputTransform           string             `json:"output_transform,omitempty"`
	SubWorkflowID             string             `json:"sub_workflow_id,omitempty"`
	MaxNestingDepth           int                `json:"max_nesting_depth,omitempty"`
	EventKey                  string             `json:"event_key,omitempty"`
	EventTimeoutSecs          int                `json:"event_timeout_secs,omitempty"`
	EventNotifyURL            string             `json:"event_notify_url,omitempty"`
	SleepDurationSecs         int                `json:"sleep_duration_secs,omitempty"`
	EventEmitKey              string             `json:"event_emit_key,omitempty"`
	ConcurrencyKey            string             `json:"concurrency_key,omitempty"`
	ResourceClass             string             `json:"resource_class,omitempty"`
	CostGateThresholdMicrousd int64              `json:"cost_gate_threshold_microusd,omitempty"`
	CostGateTimeoutSecs       int                `json:"cost_gate_timeout_secs,omitempty"`
	CostGateDefaultAction     string             `json:"cost_gate_default_action,omitempty"`
	CreatedAt                 time.Time          `json:"created_at"`
}

// WorkflowRun represents an execution instance of a workflow.
type WorkflowRun struct {
	ID                  string            `json:"id"`
	WorkflowID          string            `json:"workflow_id"`
	ProjectID           string            `json:"project_id"`
	Tags                map[string]string `json:"tags,omitempty"`
	Status              WorkflowRunStatus `json:"status"`
	TriggeredBy         string            `json:"triggered_by"`
	WorkflowVersion     int               `json:"workflow_version"`
	MaxParallelSteps    int               `json:"max_parallel_steps,omitempty"`
	Payload             json.RawMessage   `json:"payload,omitempty"`
	Error               string            `json:"error,omitempty"`
	StartedAt           *time.Time        `json:"started_at,omitempty"`
	FinishedAt          *time.Time        `json:"finished_at,omitempty"`
	ExpiresAt           *time.Time        `json:"expires_at,omitempty"`
	RetryOfRunID        string            `json:"retry_of_run_id,omitempty"`
	ParentWorkflowRunID string            `json:"parent_workflow_run_id,omitempty"`
	ParentStepRunID     string            `json:"parent_step_run_id,omitempty"`
	WorkflowVersionID   string            `json:"workflow_version_id,omitempty"`
	WorkflowSnapshotID  string            `json:"workflow_snapshot_id,omitempty"`
	CreatedBy           string            `json:"created_by,omitempty"`
	TraceContext        map[string]string `json:"trace_context,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
}

// WorkflowStepRun represents execution of a single step within a workflow run.
type WorkflowStepRun struct {
	ID             string          `json:"id"`
	WorkflowRunID  string          `json:"workflow_run_id"`
	WorkflowStepID string          `json:"workflow_step_id"`
	StepRef        string          `json:"step_ref"`
	JobRunID       string          `json:"job_run_id,omitempty"`
	Attempt        int             `json:"attempt"`
	Status         StepRunStatus   `json:"status"`
	DepsCompleted  int             `json:"deps_completed"`
	DepsRequired   int             `json:"deps_required"`
	Output         json.RawMessage `json:"output,omitempty"`
	Error          string          `json:"error,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// APIKey represents an API key as returned by the REST API.
type APIKey struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	Name      string     `json:"name"`
	KeyPrefix string     `json:"key_prefix"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// EventTrigger represents a durable wait for an external event signal.
type EventTrigger struct {
	ID                string          `json:"id"`
	EventKey          string          `json:"event_key"`
	ProjectID         string          `json:"project_id"`
	SourceType        string          `json:"source_type"`
	WorkflowRunID     string          `json:"workflow_run_id,omitempty"`
	WorkflowStepRunID string          `json:"workflow_step_run_id,omitempty"`
	JobRunID          string          `json:"job_run_id,omitempty"`
	Status            string          `json:"status"`
	RequestPayload    json.RawMessage `json:"request_payload,omitempty"`
	ResponsePayload   json.RawMessage `json:"response_payload,omitempty"`
	TimeoutSecs       int             `json:"timeout_secs"`
	RequestedAt       time.Time       `json:"requested_at"`
	ReceivedAt        *time.Time      `json:"received_at,omitempty"`
	ExpiresAt         time.Time       `json:"expires_at"`
	Error             string          `json:"error,omitempty"`
	NotifyURL         string          `json:"notify_url,omitempty"`
	NotifyStatus      string          `json:"notify_status,omitempty"`
	TriggerType       string          `json:"trigger_type,omitempty"`
	SentBy            string          `json:"sent_by,omitempty"`
}

// Environment represents a project environment.
type Environment struct {
	ID         string            `json:"id"`
	ProjectID  string            `json:"project_id"`
	Name       string            `json:"name"`
	Slug       string            `json:"slug"`
	ParentID   string            `json:"parent_id,omitempty"`
	IsStandard bool              `json:"is_standard"`
	Variables  map[string]string `json:"variables,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// Project represents a project.
type Project struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProjectRole defines a named set of permissions within a project.
type ProjectRole struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Permissions  []string  `json:"permissions"`
	ParentRoleID string    `json:"parent_role_id,omitempty"`
	IsSystem     bool      `json:"is_system"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProjectMemberRole links a user to a role within a project.
type ProjectMemberRole struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	UserID    string    `json:"user_id"`
	RoleID    string    `json:"role_id"`
	GrantedBy string    `json:"granted_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// AuditEvent records sensitive control-plane actions.
type AuditEvent struct {
	ID           string          `json:"id"`
	ProjectID    string          `json:"project_id"`
	ActorID      string          `json:"actor_id"`
	ActorType    string          `json:"actor_type"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	Details      json.RawMessage `json:"details,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// StepOverride allows selectively enabling or disabling steps at trigger time.
type StepOverride struct {
	StepRef string `json:"step_ref"`
	Enabled bool   `json:"enabled"`
}

// Webhook represents an outbound webhook subscription.
type Webhook struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"project_id"`
	URL        string     `json:"url"`
	Events     []string   `json:"events"`
	Secret     string     `json:"secret,omitempty"`
	Active     bool       `json:"active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastSentAt *time.Time `json:"last_sent_at,omitempty"`
}

// WebhookDelivery represents a single webhook delivery attempt.
type WebhookDelivery struct {
	ID           string     `json:"id"`
	WebhookID    string     `json:"webhook_id"`
	EventType    string     `json:"event_type"`
	Status       string     `json:"status"`
	StatusCode   int        `json:"status_code,omitempty"`
	AttemptCount int        `json:"attempt_count"`
	Error        string     `json:"error,omitempty"`
	RequestedAt  time.Time  `json:"requested_at"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
}

// EventSource represents an external event source feeding the project.
type EventSource struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	Name      string          `json:"name"`
	Slug      string          `json:"slug"`
	Type      string          `json:"type"`
	Config    json.RawMessage `json:"config,omitempty"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// JobGroup represents a logical grouping of jobs.
type JobGroup struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	Paused      bool      `json:"paused"`
	JobCount    int       `json:"job_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// JobGroupStats holds aggregate execution metrics for a job group.
type JobGroupStats struct {
	GroupID    string `json:"group_id"`
	JobCount   int    `json:"job_count"`
	RunsTotal  int64  `json:"runs_total"`
	RunsFailed int64  `json:"runs_failed"`
	RunsActive int64  `json:"runs_active"`
}

// NotificationChannel represents a notification delivery channel.
type NotificationChannel struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Config    json.RawMessage `json:"config,omitempty"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// LogDrain represents a destination for streaming logs.
type LogDrain struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Config    json.RawMessage `json:"config,omitempty"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// JobHealth captures health status for a job.
type JobHealth struct {
	JobID         string    `json:"job_id"`
	Status        string    `json:"status"`
	LastRunStatus string    `json:"last_run_status,omitempty"`
	LastRunAt     time.Time `json:"last_run_at,omitzero"`
	SuccessRate   float64   `json:"success_rate,omitempty"`
	P95DurationMS int64     `json:"p95_duration_ms,omitempty"`
}

// JobDependency represents a dependency edge between two jobs.
type JobDependency struct {
	ID        string    `json:"id"`
	JobID     string    `json:"job_id"`
	DependsOn string    `json:"depends_on"`
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// WorkflowVersion represents a versioned workflow definition.
type WorkflowVersion struct {
	WorkflowID string    `json:"workflow_id"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  string    `json:"created_by,omitempty"`
}

// WorkflowDiff represents the difference between two workflow versions.
type WorkflowDiff struct {
	WorkflowID string          `json:"workflow_id"`
	From       int             `json:"from"`
	To         int             `json:"to"`
	Changes    json.RawMessage `json:"changes,omitempty"`
}

// WorkflowPolicy represents the run-time policy for a workflow.
type WorkflowPolicy struct {
	WorkflowID string          `json:"workflow_id"`
	Policy     json.RawMessage `json:"policy"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// RunOutput represents output produced by a run.
type RunOutput struct {
	ID        string          `json:"id"`
	RunID     string          `json:"run_id"`
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// RunToolCall represents a tool invocation within a run.
type RunToolCall struct {
	ID         string          `json:"id"`
	RunID      string          `json:"run_id"`
	Tool       string          `json:"tool"`
	Args       json.RawMessage `json:"args,omitempty"`
	Result     json.RawMessage `json:"result,omitempty"`
	DurationMS int64           `json:"duration_ms,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// RunUsage captures resource usage for a run.
type RunUsage struct {
	RunID         string  `json:"run_id"`
	TokensInput   int64   `json:"tokens_input,omitempty"`
	TokensOutput  int64   `json:"tokens_output,omitempty"`
	CostUSD       float64 `json:"cost_usd,omitempty"`
	DurationMS    int64   `json:"duration_ms,omitempty"`
	CPUSeconds    float64 `json:"cpu_seconds,omitempty"`
	MemoryMBHours float64 `json:"memory_mb_hours,omitempty"`
}

// RunCheckpoint represents a checkpoint in a run.
type RunCheckpoint struct {
	ID        string          `json:"id"`
	RunID     string          `json:"run_id"`
	Name      string          `json:"name,omitempty"`
	State     json.RawMessage `json:"state,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// DLQRun represents a run that has been moved to the dead letter queue.
type DLQRun struct {
	ID         string    `json:"id"`
	RunID      string    `json:"run_id"`
	JobID      string    `json:"job_id,omitempty"`
	Reason     string    `json:"reason"`
	FailedAt   time.Time `json:"failed_at"`
	AttemptCnt int       `json:"attempt_count"`
}

// UsagePeriod represents resource usage and billing for a billing period.
type UsagePeriod struct {
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	Runs             int64     `json:"runs"`
	WorkflowRuns     int64     `json:"workflow_runs"`
	ComputeMinutes   float64   `json:"compute_minutes"`
	StorageMBHours   float64   `json:"storage_mb_hours,omitempty"`
	EgressMB         float64   `json:"egress_mb,omitempty"`
	TokensInput      int64     `json:"tokens_input,omitempty"`
	TokensOutput     int64     `json:"tokens_output,omitempty"`
	CostUSD          float64   `json:"cost_usd"`
	IncludedQuotaPct float64   `json:"included_quota_pct,omitempty"`
}

// CostsAnalytics summarises spend for the analytics period.
type CostsAnalytics struct {
	PeriodHours int                    `json:"period_hours"`
	TotalUSD    float64                `json:"total_usd"`
	ByJob       []CostByJob            `json:"by_job,omitempty"`
	ByCategory  map[string]float64     `json:"by_category,omitempty"`
	Series      []CostsTimeseriesPoint `json:"series,omitempty"`
}

// CostByJob is a per-job cost breakdown.
type CostByJob struct {
	JobID   string  `json:"job_id"`
	JobSlug string  `json:"job_slug"`
	Runs    int64   `json:"runs"`
	USD     float64 `json:"usd"`
}

// CostsTimeseriesPoint is one bucket of the cost timeseries.
type CostsTimeseriesPoint struct {
	BucketStart time.Time `json:"bucket_start"`
	USD         float64   `json:"usd"`
}

// ReliabilityAnalytics summarises reliability over the analytics period.
type ReliabilityAnalytics struct {
	PeriodHours       int     `json:"period_hours"`
	SuccessRate       float64 `json:"success_rate"`
	AvgDurationSecs   float64 `json:"avg_duration_secs"`
	P95DurationSecs   float64 `json:"p95_duration_secs"`
	RetriedRunPercent float64 `json:"retried_run_percent,omitempty"`
}

// TopFailingJob represents a job with elevated failure rate.
type TopFailingJob struct {
	JobID       string  `json:"job_id"`
	JobSlug     string  `json:"job_slug"`
	TotalRuns   int64   `json:"total_runs"`
	FailedRuns  int64   `json:"failed_runs"`
	FailureRate float64 `json:"failure_rate"`
}

// TeamPolicy represents an RBAC policy granting permissions to a team.
type TeamPolicy struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	ResourcePattern string    `json:"resource_pattern,omitempty"`
	TagPattern      string    `json:"tag_pattern,omitempty"`
	Permissions     []string  `json:"permissions"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// WorkerInfo describes a connected worker, returned by the workers endpoint.
type WorkerInfo struct {
	ID            string    `json:"id"`
	Name          string    `json:"name,omitempty"`
	ProjectID     string    `json:"project_id,omitempty"`
	Queues        []string  `json:"queues,omitempty"`
	Concurrency   int       `json:"concurrency,omitempty"`
	Status        string    `json:"status"`
	Version       string    `json:"version,omitempty"`
	RemoteAddr    string    `json:"remote_addr,omitempty"`
	ConnectedAt   time.Time `json:"connected_at"`
	LastHeartbeat time.Time `json:"last_heartbeat,omitzero"`
	ActiveTasks   int       `json:"active_tasks"`
}
