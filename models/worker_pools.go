package models

// a task is single unit of work submitted to worker pool
type Task struct {
	ID      string
	Execute func() (any, error)
}

// result is something that comes post the execution of the Task submitted
type Result struct {
	ID    string
	Value any
	Err   error // each execution have it's own error making it independent of other executions
}
