package config

import (
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

	/* for name, workflow := range wc.Workflows {
		fmt.Printf("%s - %s, %s\n", name, workflow.Identifier, workflow.Description)
		for _, node := range workflow.Nodes {
			fmt.Printf("mod: %s | op: %s | args: %v | type: %s\n", node.Module, node.Operator, node.Args, node.Type)
		}
	} */

	return wc
}
