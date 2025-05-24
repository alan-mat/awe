package config

const (
	NodeTypeLinear      = "default"
	NodeTypeLoop        = "loop"
	NodeTypeConditional = "conditional"
	NodeTypeBranching   = "parallel"
)

type WorkflowNodeType string

var workflowNodeTypeMap = map[string]WorkflowNodeType{
	"default":     NodeTypeLinear,
	"loop":        NodeTypeLoop,
	"conditional": NodeTypeConditional,
	"parallel":    NodeTypeBranching,
}

type WorkflowNode struct {
	Module   string           `yaml:"module"`
	Operator string           `yaml:"operator"`
	Type     WorkflowNodeType `yaml:"type"`
	Args     map[string]any   `yaml:"args"`

	Nodes    []WorkflowNode   `yaml:"nodes"`
	Routes   []WorkflowRoute  `yaml:"routes"`
	Branches []WorkflowBranch `yaml:"branches"`
}

type WorkflowRoute struct {
	Key         string         `yaml:"key"`
	Description string         `yaml:"description"`
	Nodes       []WorkflowNode `yaml:"nodes"`
}

type WorkflowBranch struct {
	Name  string         `yaml:"name"`
	Nodes []WorkflowNode `yaml:"nodes"`
}

type Workflow struct {
	Identifier     string `yaml:"name"`
	Description    string `yaml:"description"`
	CollectionName string `yaml:"collection"`

	Nodes []WorkflowNode `yaml:"nodes"`
}

type WorkflowConfig struct {
	Workflows map[string]Workflow `yaml:"workflows"`
}
