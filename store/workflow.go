package store

import (
	"context"
	"time"
)

type WorkflowStatus string

const (
	StatusPending  WorkflowStatus = "pending"
	StatusRunning  WorkflowStatus = "running"
	StatusCompleted WorkflowStatus = "completed"
	StatusFailed   WorkflowStatus = "failed"
	StatusCancelled WorkflowStatus = "cancelled"
)

type TaskDefinition struct {
	ID           string   `json:"id"`
	Execute      string   `json:"execute"`
	Dependencies []string `json:"dependencies,omitempty"`
}

type TaskLog struct {
	Stream    string    `json:"stream"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskStatus struct {
	ID        string       `json:"id"`
	State     string       `json:"state"`
	Value     any          `json:"value,omitempty"`
	Error     string       `json:"error,omitempty"`
	StartedAt *time.Time   `json:"started_at,omitempty"`
	EndedAt   *time.Time   `json:"ended_at,omitempty"`
}

type Workflow struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Status       WorkflowStatus        `json:"status"`
	Tasks        []TaskDefinition      `json:"tasks"`
	TaskStatuses map[string]TaskStatus `json:"task_statuses,omitempty"`
	Error        string                `json:"error,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	StartedAt    *time.Time            `json:"started_at,omitempty"`
	EndedAt      *time.Time            `json:"ended_at,omitempty"`
}

type SubmitWorkflowRequest struct {
	Name  string           `json:"name"`
	Tasks []TaskDefinition `json:"tasks"`
}

type WorkflowStore interface {
	CreateWorkflow(req SubmitWorkflowRequest) Workflow
	GetWorkflow(id string) (Workflow, bool)
	ListWorkflows() []Workflow
	UpdateWorkflow(id string, status WorkflowStatus, errMsg string)
	UpdateTaskStatus(workflowID, taskID, state string, value any, errMsg string)
	CancelWorkflow(id string) bool
	ExecuteWorkflow(ctx context.Context, id string) error
	AppendTaskLog(workflowID, taskID, stream, content string)
	GetTaskLogs(workflowID, taskID string) []TaskLog
	GetAllTaskLogs(workflowID string) map[string][]TaskLog
}
