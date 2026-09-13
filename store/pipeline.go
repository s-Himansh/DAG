package store

import (
	"os"

	"gopkg.in/yaml.v3"
)

type PipelineFile struct {
	Name    string         `yaml:"name"`
	Tasks   []TaskFile     `yaml:"tasks"`
}

type TaskFile struct {
	ID           string   `yaml:"id"`
	Execute      string   `yaml:"execute"`
	Dependencies []string `yaml:"depends_on,omitempty"`
}

func ParsePipelineFile(path string) (*PipelineFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pipeline PipelineFile
	if err := yaml.Unmarshal(data, &pipeline); err != nil {
		return nil, err
	}

	return &pipeline, nil
}

func PipelineToRequest(p *PipelineFile) SubmitWorkflowRequest {
	var tasks []TaskDefinition
	for _, t := range p.Tasks {
		tasks = append(tasks, TaskDefinition{
			ID:           t.ID,
			Execute:      t.Execute,
			Dependencies: t.Dependencies,
		})
	}

	return SubmitWorkflowRequest{
		Name:  p.Name,
		Tasks: tasks,
	}
}
