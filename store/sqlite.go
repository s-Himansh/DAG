package store

import (
	"context"
	"database/sql"
	"dag/dag"
	"dag/executor"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS workflows (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			error TEXT,
			created_at DATETIME NOT NULL,
			started_at DATETIME,
			ended_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS task_definitions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workflow_id TEXT NOT NULL,
			task_id TEXT NOT NULL,
			execute TEXT NOT NULL DEFAULT 'execute',
			dependencies TEXT,
			FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS task_statuses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workflow_id TEXT NOT NULL,
			task_id TEXT NOT NULL,
			state TEXT NOT NULL DEFAULT 'pending',
			value TEXT,
			error TEXT,
			started_at DATETIME,
			ended_at DATETIME,
			UNIQUE(workflow_id, task_id),
			FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS task_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workflow_id TEXT NOT NULL,
			task_id TEXT NOT NULL,
			stream TEXT NOT NULL DEFAULT 'stdout',
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
		)`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}

	return nil
}

func (s *SQLiteStore) CreateWorkflow(req SubmitWorkflowRequest) Workflow {
	id := uuid.New().String()[:8]
	now := time.Now()

	tx, err := s.db.Begin()
	if err != nil {
		return Workflow{}
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO workflows (id, name, status, created_at) VALUES (?, ?, ?, ?)`,
		id, req.Name, "pending", now,
	)
	if err != nil {
		return Workflow{}
	}

	for _, t := range req.Tasks {
		deps, _ := json.Marshal(t.Dependencies)
		_, err = tx.Exec(
			`INSERT INTO task_definitions (workflow_id, task_id, execute, dependencies) VALUES (?, ?, ?, ?)`,
			id, t.ID, t.Execute, string(deps),
		)
		if err != nil {
			return Workflow{}
		}

		_, err = tx.Exec(
			`INSERT INTO task_statuses (workflow_id, task_id, state) VALUES (?, ?, ?)`,
			id, t.ID, "pending",
		)
		if err != nil {
			return Workflow{}
		}
	}

	if err := tx.Commit(); err != nil {
		return Workflow{}
	}

	w := Workflow{
		ID:        id,
		Name:      req.Name,
		Status:    StatusPending,
		Tasks:     req.Tasks,
		CreatedAt: now,
	}
	w.TaskStatuses = make(map[string]TaskStatus)
	for _, t := range req.Tasks {
		w.TaskStatuses[t.ID] = TaskStatus{
			ID:    t.ID,
			State: "pending",
		}
	}
	return w
}

func (s *SQLiteStore) GetWorkflow(id string) (Workflow, bool) {
	var w Workflow
	var startedAt, endedAt sql.NullTime
	var errMsg sql.NullString

	err := s.db.QueryRow(
		`SELECT id, name, status, error, created_at, started_at, ended_at FROM workflows WHERE id = ?`,
		id,
	).Scan(&w.ID, &w.Name, &w.Status, &errMsg, &w.CreatedAt, &startedAt, &endedAt)
	if err != nil {
		return Workflow{}, false
	}

	if errMsg.Valid {
		w.Error = errMsg.String
	}
	if startedAt.Valid {
		w.StartedAt = &startedAt.Time
	}
	if endedAt.Valid {
		w.EndedAt = &endedAt.Time
	}

	rows, err := s.db.Query(
		`SELECT task_id, execute, dependencies FROM task_definitions WHERE workflow_id = ?`,
		id,
	)
	if err != nil {
		return Workflow{}, false
	}
	defer rows.Close()

	for rows.Next() {
		var t TaskDefinition
		var deps string
		if err := rows.Scan(&t.ID, &t.Execute, &deps); err != nil {
			continue
		}
		json.Unmarshal([]byte(deps), &t.Dependencies)
		w.Tasks = append(w.Tasks, t)
	}

	w.TaskStatuses = make(map[string]TaskStatus)
	rows2, err := s.db.Query(
		`SELECT task_id, state, value, error, started_at, ended_at FROM task_statuses WHERE workflow_id = ?`,
		id,
	)
	if err != nil {
		return Workflow{}, false
	}
	defer rows2.Close()

	for rows2.Next() {
		var ts TaskStatus
		var value, errMsg sql.NullString
		var tsStarted, tsEnded sql.NullTime
		if err := rows2.Scan(&ts.ID, &ts.State, &value, &errMsg, &tsStarted, &tsEnded); err != nil {
			continue
		}
		if value.Valid {
			ts.Value = value.String
		}
		if errMsg.Valid {
			ts.Error = errMsg.String
		}
		if tsStarted.Valid {
			ts.StartedAt = &tsStarted.Time
		}
		if tsEnded.Valid {
			ts.EndedAt = &tsEnded.Time
		}
		w.TaskStatuses[ts.ID] = ts
	}

	return w, true
}

func (s *SQLiteStore) ListWorkflows() []Workflow {
	rows, err := s.db.Query(
		`SELECT id, name, status, created_at, started_at, ended_at FROM workflows ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var workflows []Workflow
	for rows.Next() {
		var w Workflow
		var startedAt, endedAt sql.NullTime
		if err := rows.Scan(&w.ID, &w.Name, &w.Status, &w.CreatedAt, &startedAt, &endedAt); err != nil {
			continue
		}
		if startedAt.Valid {
			w.StartedAt = &startedAt.Time
		}
		if endedAt.Valid {
			w.EndedAt = &endedAt.Time
		}

		taskRows, err := s.db.Query(
			`SELECT task_id, state, value, error FROM task_statuses WHERE workflow_id = ?`,
			w.ID,
		)
		if err == nil {
			w.TaskStatuses = make(map[string]TaskStatus)
			for taskRows.Next() {
				var ts TaskStatus
				var value, errMsg sql.NullString
				if err := taskRows.Scan(&ts.ID, &ts.State, &value, &errMsg); err != nil {
					continue
				}
				if value.Valid {
					ts.Value = value.String
				}
				if errMsg.Valid {
					ts.Error = errMsg.String
				}
				w.TaskStatuses[ts.ID] = ts
			}
			taskRows.Close()
		}

		defRows, err := s.db.Query(
			`SELECT task_id, dependencies FROM task_definitions WHERE workflow_id = ?`,
			w.ID,
		)
		if err == nil {
			for defRows.Next() {
				var t TaskDefinition
				var deps string
				if err := defRows.Scan(&t.ID, &deps); err != nil {
					continue
				}
				json.Unmarshal([]byte(deps), &t.Dependencies)
				w.Tasks = append(w.Tasks, t)
			}
			defRows.Close()
		}

		workflows = append(workflows, w)
	}

	return workflows
}

func (s *SQLiteStore) UpdateWorkflow(id string, status WorkflowStatus, errMsg string) {
	now := time.Now()
	switch status {
	case StatusRunning:
		s.db.Exec(`UPDATE workflows SET status = ?, started_at = ? WHERE id = ?`, status, now, id)
	case StatusCompleted, StatusFailed, StatusCancelled:
		s.db.Exec(`UPDATE workflows SET status = ?, ended_at = ?, error = ? WHERE id = ?`, status, now, errMsg, id)
	default:
		s.db.Exec(`UPDATE workflows SET status = ? WHERE id = ?`, status, id)
	}
}

func (s *SQLiteStore) UpdateTaskStatus(workflowID, taskID, state string, value any, errMsg string) {
	now := time.Now()
	var valueStr *string
	if value != nil {
		v := fmt.Sprintf("%v", value)
		valueStr = &v
	}

	switch state {
	case "running":
		s.db.Exec(
			`UPDATE task_statuses SET state = ?, value = ?, error = ?, started_at = ? WHERE workflow_id = ? AND task_id = ?`,
			state, valueStr, errMsg, now, workflowID, taskID,
		)
	case "completed", "failed", "skipped":
		s.db.Exec(
			`UPDATE task_statuses SET state = ?, value = ?, error = ?, ended_at = ? WHERE workflow_id = ? AND task_id = ?`,
			state, valueStr, errMsg, now, workflowID, taskID,
		)
	default:
		s.db.Exec(
			`UPDATE task_statuses SET state = ?, value = ?, error = ? WHERE workflow_id = ? AND task_id = ?`,
			state, valueStr, errMsg, workflowID, taskID,
		)
	}
}

func (s *SQLiteStore) CancelWorkflow(id string) bool {
	result, err := s.db.Exec(
		`UPDATE workflows SET status = ?, ended_at = ? WHERE id = ? AND (status = ? OR status = ?)`,
		"cancelled", time.Now(), id, "pending", "running",
	)
	if err != nil {
		return false
	}
	affected, _ := result.RowsAffected()
	return affected > 0
}

func getTaskDuration(taskID string) time.Duration {
	durations := map[string]time.Duration{
		"checkout":         2 * time.Second,
		"install-deps":    3 * time.Second,
		"lint":             2 * time.Second,
		"typecheck":        2 * time.Second,
		"unit-tests":       4 * time.Second,
		"build":            5 * time.Second,
		"e2e-chrome":       4 * time.Second,
		"e2e-firefox":      4 * time.Second,
		"lighthouse":       3 * time.Second,
		"deploy-staging":   3 * time.Second,
		"smoke-tests":      2 * time.Second,
		"deploy-production": 3 * time.Second,
		"notify-slack":     1 * time.Second,
	}
	if d, ok := durations[taskID]; ok {
		return d
	}
	return 2 * time.Second
}

func (s *SQLiteStore) ExecuteWorkflow(ctx context.Context, id string) error {
	w, ok := s.GetWorkflow(id)
	if !ok {
		return fmt.Errorf("workflow not found")
	}

	s.UpdateWorkflow(id, StatusRunning, "")

	shellExec := executor.New(
		executor.WithTimeout(10*time.Minute),
		executor.WithLogFunc(func(taskID, stream, content string) {
			s.AppendTaskLog(id, taskID, stream, content)
		}),
	)

	dagBuilder := dag.NewBuilder()

	for _, t := range w.Tasks {
		taskID := t.ID
		command := t.Execute
		dagBuilder.AddTask(taskID, func() (any, error) {
			s.UpdateTaskStatus(id, taskID, "running", nil, "")

			result, err := shellExec.Run(ctx, taskID, command)
			if err != nil {
				s.UpdateTaskStatus(id, taskID, "failed", nil, err.Error())
				return nil, err
			}

			if result.ExitCode != 0 {
				errMsg := fmt.Sprintf("exit code %d: %s", result.ExitCode, result.Error)
				s.UpdateTaskStatus(id, taskID, "failed", nil, errMsg)
				return nil, fmt.Errorf("task %s failed: %s", taskID, errMsg)
			}

			s.UpdateTaskStatus(id, taskID, "completed", result.Output, "")
			return result.Output, nil
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

func (s *SQLiteStore) AppendTaskLog(workflowID, taskID, stream, content string) {
	s.db.Exec(
		`INSERT INTO task_logs (workflow_id, task_id, stream, content) VALUES (?, ?, ?, ?)`,
		workflowID, taskID, stream, content,
	)
}

func (s *SQLiteStore) GetTaskLogs(workflowID, taskID string) []TaskLog {
	rows, err := s.db.Query(
		`SELECT stream, content, created_at FROM task_logs WHERE workflow_id = ? AND task_id = ? ORDER BY id`,
		workflowID, taskID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var logs []TaskLog
	for rows.Next() {
		var l TaskLog
		if err := rows.Scan(&l.Stream, &l.Content, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs
}

func (s *SQLiteStore) GetAllTaskLogs(workflowID string) map[string][]TaskLog {
	rows, err := s.db.Query(
		`SELECT task_id, stream, content, created_at FROM task_logs WHERE workflow_id = ? ORDER BY id`,
		workflowID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make(map[string][]TaskLog)
	for rows.Next() {
		var taskID string
		var l TaskLog
		if err := rows.Scan(&taskID, &l.Stream, &l.Content, &l.CreatedAt); err != nil {
			continue
		}
		result[taskID] = append(result[taskID], l)
	}
	return result
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
