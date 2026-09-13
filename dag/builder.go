package dag

import (
	"dag/models"
	"fmt"
)

type Builder struct {
	dag *models.DAG
}

func NewBuilder() *Builder {
	return &Builder{dag: models.NewDAG()}
}

func (b *Builder) AddTask(id string, execute func() (any, error)) *Builder {
	task := &models.Task{ID: id, Execute: execute}
	node := models.NewNode(id, task)
	b.dag.AddNode(node)
	return b
}

func (b *Builder) DependsOn(taskID string, dependencies ...string) *Builder {
	for _, dep := range dependencies {
		b.dag.AddEdge(dep, taskID)
	}
	return b
}

func (b *Builder) Build() (*models.DAG, error) {
	if err := validate(b.dag); err != nil {
		return nil, err
	}
	return b.dag, nil
}

func validate(d *models.DAG) error {
	if len(d.Nodes) == 0 {
		return fmt.Errorf("dag has no nodes")
	}

	inDegree := make(map[string]int)
	for _, node := range d.Nodes {
		inDegree[node.ID] = 0
	}

	for _, edge := range d.Edges {
		inDegree[edge.To]++
	}

	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	visited := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		visited++

		for _, edge := range d.Edges {
			if edge.From == current {
				inDegree[edge.To]--
				if inDegree[edge.To] == 0 {
					queue = append(queue, edge.To)
				}
			}
		}
	}

	if visited != len(d.Nodes) {
		return fmt.Errorf("dag contains a cycle")
	}

	return nil
}
