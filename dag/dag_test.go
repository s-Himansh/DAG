package dag

import (
	"context"
	"dag/models"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecutorFanOutFanIn(t *testing.T) {
	var counter int64

	d, err := NewBuilder().
		AddTask("start", func() (any, error) {
			return atomic.AddInt64(&counter, 1), nil
		}).
		AddTask("fan1", func() (any, error) {
			return atomic.AddInt64(&counter, 1), nil
		}).
		AddTask("fan2", func() (any, error) {
			return atomic.AddInt64(&counter, 1), nil
		}).
		AddTask("fan3", func() (any, error) {
			return atomic.AddInt64(&counter, 1), nil
		}).
		AddTask("end", func() (any, error) {
			return atomic.AddInt64(&counter, 1), nil
		}).
		DependsOn("fan1", "start").
		DependsOn("fan2", "start").
		DependsOn("fan3", "start").
		DependsOn("end", "fan1").
		DependsOn("end", "fan2").
		DependsOn("end", "fan3").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exec := NewExecutor(d, 3)
	results, err := exec.Execute(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}

	for _, id := range []string{"start", "fan1", "fan2", "fan3", "end"} {
		if results[id] == nil || results[id].Err != nil {
			t.Errorf("task %s should have completed successfully", id)
		}
	}

	if results["start"].Value.(int64) != 1 {
		t.Errorf("start should run first, got %v", results["start"].Value)
	}
}

func TestExecutorContextCancellation(t *testing.T) {
	d, err := NewBuilder().
		AddTask("a", func() (any, error) {
			time.Sleep(100 * time.Millisecond)
			return "a", nil
		}).
		AddTask("b", func() (any, error) {
			time.Sleep(100 * time.Millisecond)
			return "b", nil
		}).
		DependsOn("b", "a").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	exec := NewExecutor(d, 1)
	_, err = exec.Execute(ctx)

	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestExecutorTaskFailure(t *testing.T) {
	d, err := NewBuilder().
		AddTask("a", func() (any, error) {
			return nil, fmt.Errorf("task a failed")
		}).
		AddTask("b", func() (any, error) {
			return "b", nil
		}).
		DependsOn("b", "a").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exec := NewExecutor(d, 1)
	_, err = exec.Execute(context.Background())

	if err == nil {
		t.Fatal("expected task failure error")
	}
}

func TestBuilderSimpleDAG(t *testing.T) {
	d, err := NewBuilder().
		AddTask("a", func() (any, error) { return "a", nil }).
		AddTask("b", func() (any, error) { return "b", nil }).
		AddTask("c", func() (any, error) { return "c", nil }).
		DependsOn("b", "a").
		DependsOn("c", "b").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(d.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(d.Nodes))
	}

	if len(d.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(d.Edges))
	}
}

func TestBuilderCycleDetection(t *testing.T) {
	_, err := NewBuilder().
		AddTask("a", func() (any, error) { return nil, nil }).
		AddTask("b", func() (any, error) { return nil, nil }).
		DependsOn("b", "a").
		DependsOn("a", "b").
		Build()

	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestBuilderEmptyDAG(t *testing.T) {
	_, err := NewBuilder().Build()
	if err == nil {
		t.Fatal("expected error for empty DAG")
	}
}

func TestTopologicalSort(t *testing.T) {
	d := models.NewDAG()
	d.AddNode(models.NewNode("a", &models.Task{ID: "a"}))
	d.AddNode(models.NewNode("b", &models.Task{ID: "b"}))
	d.AddNode(models.NewNode("c", &models.Task{ID: "c"}))
	d.AddNode(models.NewNode("d", &models.Task{ID: "d"}))
	d.AddEdge("a", "b")
	d.AddEdge("a", "c")
	d.AddEdge("b", "d")
	d.AddEdge("c", "d")

	order, err := TopologicalSort(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 4 {
		t.Fatalf("expected 4 nodes in order, got %d", len(order))
	}

	indexOf := func(id string) int {
		for i, v := range order {
			if v == id {
				return i
			}
		}
		return -1
	}

	if indexOf("a") > indexOf("b") {
		t.Error("a should come before b")
	}
	if indexOf("a") > indexOf("c") {
		t.Error("a should come before c")
	}
	if indexOf("b") > indexOf("d") {
		t.Error("b should come before d")
	}
	if indexOf("c") > indexOf("d") {
		t.Error("c should come before d")
	}
}

func TestTopologicalSortCycle(t *testing.T) {
	d := models.NewDAG()
	d.AddNode(models.NewNode("a", &models.Task{ID: "a"}))
	d.AddNode(models.NewNode("b", &models.Task{ID: "b"}))
	d.AddEdge("a", "b")
	d.AddEdge("b", "a")

	_, err := TopologicalSort(d)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestExecutionLevels(t *testing.T) {
	d := models.NewDAG()
	d.AddNode(models.NewNode("a", &models.Task{ID: "a"}))
	d.AddNode(models.NewNode("b", &models.Task{ID: "b"}))
	d.AddNode(models.NewNode("c", &models.Task{ID: "c"}))
	d.AddNode(models.NewNode("d", &models.Task{ID: "d"}))
	d.AddEdge("a", "b")
	d.AddEdge("a", "c")
	d.AddEdge("b", "d")
	d.AddEdge("c", "d")

	levels, err := GetExecutionLevels(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(levels) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(levels))
	}

	if len(levels[0]) != 1 || levels[0][0] != "a" {
		t.Errorf("level 0 should be [a], got %v", levels[0])
	}
	if len(levels[1]) != 2 {
		t.Errorf("level 1 should have 2 nodes, got %d", len(levels[1]))
	}
	if len(levels[2]) != 1 || levels[2][0] != "d" {
		t.Errorf("level 2 should be [d], got %v", levels[2])
	}
}

func TestExecutorSimpleDAG(t *testing.T) {
	d, err := NewBuilder().
		AddTask("a", func() (any, error) { return 1, nil }).
		AddTask("b", func() (any, error) { return 2, nil }).
		AddTask("c", func() (any, error) { return 3, nil }).
		DependsOn("b", "a").
		DependsOn("c", "b").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exec := NewExecutor(d, 2)
	results, err := exec.Execute(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results["a"].Value.(int) != 1 {
		t.Errorf("expected a=1, got %v", results["a"].Value)
	}
	if results["b"].Value.(int) != 2 {
		t.Errorf("expected b=2, got %v", results["b"].Value)
	}
	if results["c"].Value.(int) != 3 {
		t.Errorf("expected c=3, got %v", results["c"].Value)
	}
}
