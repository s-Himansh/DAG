package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

type TaskResult struct {
	TaskID   string
	ExitCode int
	Output   string
	Error    string
	Duration time.Duration
}

type LogFunc func(taskID, stream, content string)

type ShellExecutor struct {
	logFn    LogFunc
	timeout  time.Duration
	workDir  string
	env      []string
}

type Option func(*ShellExecutor)

func WithTimeout(d time.Duration) Option {
	return func(e *ShellExecutor) { e.timeout = d }
}

func WithWorkDir(dir string) Option {
	return func(e *ShellExecutor) { e.workDir = dir }
}

func WithEnv(env []string) Option {
	return func(e *ShellExecutor) { e.env = env }
}

func WithLogFunc(fn LogFunc) Option {
	return func(e *ShellExecutor) { e.logFn = fn }
}

func New(opts ...Option) *ShellExecutor {
	e := &ShellExecutor{
		timeout: 10 * time.Minute,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (e *ShellExecutor) Run(ctx context.Context, taskID, command string) (*TaskResult, error) {
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	start := time.Now()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	if e.workDir != "" {
		cmd.Dir = e.workDir
	}
	if len(e.env) > 0 {
		cmd.Env = e.env
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		e.streamOutput(ctx, taskID, "stdout", stdout, &stdoutBuf)
	}()
	go func() {
		defer wg.Done()
		e.streamOutput(ctx, taskID, "stderr", stderr, &stderrBuf)
	}()

	wg.Wait()

	err = cmd.Wait()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			return &TaskResult{
				TaskID:   taskID,
				ExitCode: -1,
				Error:    fmt.Sprintf("command timed out after %s", e.timeout),
				Duration: duration,
			}, nil
		} else {
			return nil, fmt.Errorf("failed to wait for command: %w", err)
		}
	}

	return &TaskResult{
		TaskID:   taskID,
		ExitCode: exitCode,
		Output:   stdoutBuf.String(),
		Error:    stderrBuf.String(),
		Duration: duration,
	}, nil
}

func (e *ShellExecutor) streamOutput(ctx context.Context, taskID, stream string, r io.Reader, buf *bytes.Buffer) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteString("\n")

		if e.logFn != nil {
			e.logFn(taskID, stream, line+"\n")
		}
	}
}

func (e *ShellExecutor) RunWithCapture(ctx context.Context, taskID, command string) (*TaskResult, error) {
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	start := time.Now()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	if e.workDir != "" {
		cmd.Dir = e.workDir
	}
	if len(e.env) > 0 {
		cmd.Env = e.env
	}

	output, err := cmd.CombinedOutput()

	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			return &TaskResult{
				TaskID:   taskID,
				ExitCode: -1,
				Error:    fmt.Sprintf("command timed out after %s", e.timeout),
				Duration: duration,
			}, nil
		} else {
			return nil, fmt.Errorf("failed to run command: %w", err)
		}
	}

	return &TaskResult{
		TaskID:   taskID,
		ExitCode: exitCode,
		Output:   string(output),
		Duration: duration,
	}, nil
}
