package dag

import (
	"dag/models"
	"fmt"
)

func TopologicalSort(d *models.DAG) ([]string, error) {
	inDegree := make(map[string]int)
	for id := range d.GetNodes() {
		inDegree[id] = 0
	}

	for _, edge := range d.GetEdges() {
		inDegree[edge.To]++
	}

	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	result := make([]string, 0, d.NodeCount())

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, edge := range d.GetEdges() {
			if edge.From == current {
				inDegree[edge.To]--
				if inDegree[edge.To] == 0 {
					queue = append(queue, edge.To)
				}
			}
		}
	}

	if len(result) != d.NodeCount() {
		return nil, fmt.Errorf("cycle detected: processed %d of %d nodes", len(result), d.NodeCount())
	}

	return result, nil
}

func DetectCycle(d *models.DAG) bool {
	_, err := TopologicalSort(d)
	return err != nil
}

func GetExecutionLevels(d *models.DAG) ([][]string, error) {
	levels := make([][]string, 0)

	inDegree := make(map[string]int)
	for id := range d.GetNodes() {
		inDegree[id] = 0
	}

	for _, edge := range d.GetEdges() {
		inDegree[edge.To]++
	}

	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	for len(queue) > 0 {
		level := make([]string, len(queue))
		copy(level, queue)
		levels = append(levels, level)

		nextQueue := make([]string, 0)
		for _, current := range queue {
			for _, edge := range d.GetEdges() {
				if edge.From == current {
					inDegree[edge.To]--
					if inDegree[edge.To] == 0 {
						nextQueue = append(nextQueue, edge.To)
					}
				}
			}
		}
		queue = nextQueue
	}

	total := 0
	for _, level := range levels {
		total += len(level)
	}
	if total != d.NodeCount() {
		return nil, fmt.Errorf("cycle detected")
	}

	return levels, nil
}
