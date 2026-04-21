package workerpools

import (
	"context"
	"dag/models"
	"sync"
)

type Pool struct {
	workers  int
	taskCh   chan *models.Task
	resultCh chan *models.Result
	wg       sync.WaitGroup
}

func NewPool(workers, queueSize int) *Pool {
	return &Pool{workers: workers, taskCh: make(chan *models.Task, queueSize), resultCh: make(chan *models.Result, queueSize)}
}

func (p *Pool) Start(ctx context.Context) {
	for range p.workers {
		p.wg.Add(1)

		go p.worker(ctx)
	}
}

func (p *Pool) worker(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-p.taskCh:
			if !ok {
				return
			}

			res, err := task.Execute()

			p.resultCh <- &models.Result{ID: task.ID, Value: res, Err: err}
		}
	}
}

func (p *Pool) Submit(task *models.Task) {
	p.taskCh <- task
}

func (p *Pool) Results() <-chan *models.Result {
	return p.resultCh
}

// Shutdown stops accepting tasks, waits for workers, closes results
func (p *Pool) Shutdown() {
	close(p.taskCh)   // no more tasks
	p.wg.Wait()       // wait for workers to finish
	close(p.resultCh) // no more results
}
