package store

import (
	"context"
	"dag/dag"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	workflows map[string]*Workflow
	mu        sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workflows: make(map[string]*Workflow),
	}
}

func copyWorkflow(w *Workflow) *Workflow {
	cp := *w
	cp.Tasks = make([]TaskDefinition, len(w.Tasks))
	copy(cp.Tasks, w.Tasks)

	cp.TaskStatuses = make(map[string]TaskStatus, len(w.TaskStatuses))
	for k, v := range w.TaskStatuses {
		cp.TaskStatuses[k] = v
	}

	if w.StartedAt != nil {
		t := *w.StartedAt
		cp.StartedAt = &t
	}
	if w.EndedAt != nil {
		t := *w.EndedAt
		cp.EndedAt = &t
	}
	return &cp
}

func (s *MemoryStore) CreateWorkflow(req SubmitWorkflowRequest) Workflow {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()[:8]
	now := time.Now()

	taskStatuses := make(map[string]TaskStatus)
	for _, t := range req.Tasks {
		taskStatuses[t.ID] = TaskStatus{
			ID:    t.ID,
			State: "pending",
		}
	}

	w := &Workflow{
		ID:           id,
		Name:         req.Name,
		Status:       StatusPending,
		Tasks:        req.Tasks,
		TaskStatuses: taskStatuses,
		CreatedAt:    now,
	}

	s.workflows[id] = w
	return *copyWorkflow(w)
}

func (s *MemoryStore) GetWorkflow(id string) (Workflow, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workflows[id]
	if !ok {
		return Workflow{}, false
	}
	return *copyWorkflow(w), true
}

func (s *MemoryStore) ListWorkflows() []Workflow {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Workflow, 0, len(s.workflows))
	for _, w := range s.workflows {
		list = append(list, *copyWorkflow(w))
	}
	return list
}

func (s *MemoryStore) UpdateWorkflow(id string, status WorkflowStatus, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, ok := s.workflows[id]
	if !ok {
		return
	}

	w.Status = status
	now := time.Now()

	switch status {
	case StatusRunning:
		w.StartedAt = &now
	case StatusCompleted, StatusFailed, StatusCancelled:
		w.EndedAt = &now
	}

	if errMsg != "" {
		w.Error = errMsg
	}
}

func (s *MemoryStore) UpdateTaskStatus(workflowID, taskID, state string, value any, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, ok := s.workflows[workflowID]
	if !ok {
		return
	}

	ts, ok := w.TaskStatuses[taskID]
	if !ok {
		return
	}

	ts.State = state
	now := time.Now()

	switch state {
	case "running":
		ts.StartedAt = &now
	case "completed", "failed", "skipped":
		ts.EndedAt = &now
	}

	if value != nil {
		ts.Value = value
	}
	if errMsg != "" {
		ts.Error = errMsg
	}

	w.TaskStatuses[taskID] = ts
}

func (s *MemoryStore) CancelWorkflow(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, ok := s.workflows[id]
	if !ok {
		return false
	}

	if w.Status != StatusPending && w.Status != StatusRunning {
		return false
	}

	w.Status = StatusCancelled
	now := time.Now()
	w.EndedAt = &now
	return true
}

func (s *MemoryStore) ExecuteWorkflow(ctx context.Context, id string) error {
	w, ok := s.GetWorkflow(id)
	if !ok {
		return fmt.Errorf("workflow not found")
	}

	s.UpdateWorkflow(id, StatusRunning, "")

	dagBuilder := dag.NewBuilder()

	for _, t := range w.Tasks {
		taskID := t.ID
		dagBuilder.AddTask(taskID, func() (any, error) {
			s.UpdateTaskStatus(id, taskID, "running", nil, "")
			time.Sleep(100 * time.Millisecond)
			result := fmt.Sprintf("result_%s", taskID)
			s.UpdateTaskStatus(id, taskID, "completed", result, "")
			return result, nil
		})
	}

	for _, t := range w.Tasks {
		for _, dep := range t.Dependencies {
			dagBuilder.DependsOn(t.ID, dep)
		}
	}

	d, err := dagBuilder.Build()
	if err != nil {
		s.UpdateWorkflow(id, StatusFailed, err.Error())
		return err
	}

	exec := dag.NewExecutor(d, 4)
	_, err = exec.Execute(ctx)

	if err != nil {
		if ctx.Err() != nil {
			s.UpdateWorkflow(id, StatusCancelled, "workflow cancelled")
		} else {
			s.UpdateWorkflow(id, StatusFailed, err.Error())
		}
		return err
	}

	s.UpdateWorkflow(id, StatusCompleted, "")
	return nil
}
