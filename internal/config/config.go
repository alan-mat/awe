package config

import (
	"log/slog"
	"os"

	"github.com/goccy/go-yaml"
)

func ParseWorkflowConfig(path string) WorkflowConfig {
	file, err := os.ReadFile("workflows.yaml")
	if err != nil {
		panic(err)
	}

	var wc WorkflowConfig
	if err := yaml.Unmarshal(file, &wc); err != nil {
		panic(err)
	}

	slog.Info("parsed config", "value", wc)

	return wc
}
