package priority

import "dag/models"

type PriorityTask struct {
	*models.Task
	Priority int
}

type taskHeap []PriorityTask

func (th *taskHeap) Push(x any) {
	*th = append(*th, x.(PriorityTask))
}

func (th *taskHeap) Pop() any {
	old := *th

	size := len(old)

	item := old[size-1]

	*th = old[:size-1]

	return item
}

func (th *taskHeap) Less(i, j int) bool { return (*th)[i].Priority < (*th)[j].Priority }

func (th *taskHeap) Swap(i, j int) { (*th)[i], (*th)[j] = (*th)[j], (*th)[i] }

func (th *taskHeap) Len() int { return len(*th) }
