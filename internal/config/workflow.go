package config

const (
	NodeTypeLinear = "linear"
)

type WorkflowNodeType string

/* var workflowNodeTypeMap = map[string]WorkflowNodeType{
	"linear": NodeTypeLinear,
} */

type WorkflowNode struct {
	Module   string           `yaml:"module"`
	Operator string           `yaml:"op"`
	Type     WorkflowNodeType `yaml:"type"`
	Args     map[string]any   `yaml:"args"`
}

type Workflow struct {
	Identifier  string `yaml:"name"`
	Description string `yaml:"description"`

	Nodes []WorkflowNode `yaml:"nodes"`
}

type WorkflowConfig struct {
	Workflows map[string]Workflow `yaml:"workflows"`
}
