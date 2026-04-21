package workerpools

import (
	"context"
	"dag/models"
	"fmt"
	"sync"
	"testing"
)

func TestRaceCondition(t *testing.T) {
	results := make(map[string]*models.Result)

	var wg sync.WaitGroup

	ctx := context.Background()

	p := NewPool(5, 100)

	p.Start(ctx)

	for i := range 100 {
		n := i
		p.Submit(&models.Task{
			ID: fmt.Sprintf("task-%d", n),
			Execute: func() (any, error) {
				return n * n, nil
			},
		})
	}

	for range 3 {
		wg.Go(func() {
			for r := range p.Results() {
				results[r.ID] = r // concurrent map write
			}
		})
	}

	go p.Shutdown()

	wg.Wait()

	t.Logf("Got %d results", len(results))
}

// Error trace
/*
go test -race  -run TestRaceCondition

==================
WARNING: DATA RACE
Write at 0x00c0001845a0 by goroutine 14:
  runtime.mapaccess2_faststr()
      /opt/homebrew/Cellar/go/1.25.1/libexec/src/internal/runtime/maps/runtime_faststr_swiss.go:162 +0x29c
  dag/executors/pool.TestRaceCondition.func2()
      /Users/zop7945/Desktop/DAG/executors/pool/race_condition_test.go:35 +0x8c
  sync.(*WaitGroup).Go.func1()
      /opt/homebrew/Cellar/go/1.25.1/libexec/src/sync/waitgroup.go:239 +0x54

Previous write at 0x00c0001845a0 by goroutine 13:
  runtime.mapaccess2_faststr()
      /opt/homebrew/Cellar/go/1.25.1/libexec/src/internal/runtime/maps/runtime_faststr_swiss.go:162 +0x29c
  dag/executors/pool.TestRaceCondition.func2()
      /Users/zop7945/Desktop/DAG/executors/pool/race_condition_test.go:35 +0x8c
  sync.(*WaitGroup).Go.func1()
      /opt/homebrew/Cellar/go/1.25.1/libexec/src/sync/waitgroup.go:239 +0x54

Goroutine 14 (running) created at:
  sync.(*WaitGroup).Go()
      /opt/homebrew/Cellar/go/1.25.1/libexec/src/sync/waitgroup.go:237 +0x78
  dag/executors/pool.TestRaceCondition()
      /Users/zop7945/Desktop/DAG/executors/pool/race_condition_test.go:33 +0x2b4
  testing.tRunner()
      /opt/homebrew/Cellar/go/1.25.1/libexec/src/testing/testing.go:1934 +0x164
  testing.(*T).Run.gowrap1()
      /opt/homebrew/Cellar/go/1.25.1/libexec/src/testing/testing.go:1997 +0x3c

Goroutine 13 (running) created at:
  sync.(*WaitGroup).Go()
      /opt/homebrew/Cellar/go/1.25.1/libexec/src/sync/waitgroup.go:237 +0x78
  dag/executors/pool.TestRaceCondition()
      /Users/zop7945/Desktop/DAG/executors/pool/race_condition_test.go:33 +0x2b4
  testing.tRunner()
      /opt/homebrew/Cellar/go/1.25.1/libexec/src/testing/testing.go:1934 +0x164
  testing.(*T).Run.gowrap1()
      /opt/homebrew/Cellar/go/1.25.1/libexec/src/testing/testing.go:1997 +0x3c
==================
--- FAIL: TestRaceCondition (0.00s)
    race_condition_test.go:44: Got 100 results
    testing.go:1617: race detected during execution of test
FAIL
exit status 1
FAIL    dag/executors/pool      1.046s
*/
