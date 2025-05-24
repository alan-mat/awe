package registry

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/alan-mat/awe/internal/executor"
)

var (
	executorLock sync.RWMutex
	executors    = make(map[string]executor.Executor)

	workflowLock sync.RWMutex
	workflows    = make(map[string]*executor.Workflow)
)

func RegisterExecutor(name string, exec executor.Executor) error {
	executorLock.Lock()
	defer executorLock.Unlock()

	if _, exists := executors[name]; exists {
		return fmt.Errorf("failed to register, executor with name '%s' already exists", name)
	}
	slog.Info("registering executor", "name", name)
	executors[name] = exec
	return nil
}

func GetExecutor(name string) (executor.Executor, error) {
	executorLock.RLock()
	defer executorLock.RUnlock()

	exec, exists := executors[name]
	if !exists {
		return nil, fmt.Errorf("executor with name '%s' does not exist", name)
	}

	return exec, nil
}

func ListExecutors() []string {
	executorLock.RLock()
	defer executorLock.RUnlock()

	names := make([]string, 0, len(executors))
	for name := range executors {
		names = append(names, name)
	}
	return names
}

func BatchRegisterWorkflows(workflows map[string]*executor.Workflow) error {
	for name, wf := range workflows {
		err := RegisterWorkflow(name, wf)
		if err != nil {
			return err
		}
	}
	fmt.Println("registered workflows: ", ListWorkflows())
	return nil
}

func RegisterWorkflow(name string, wf *executor.Workflow) error {
	workflowLock.Lock()
	defer workflowLock.Unlock()

	if _, exists := workflows[name]; exists {
		return fmt.Errorf("failed to register, workflow with name '%s' already exists", name)
	}
	slog.Info("registering workflow", "name", name)
	workflows[name] = wf
	return nil
}

func GetWorkflow(name string) (*executor.Workflow, error) {
	workflowLock.RLock()
	defer workflowLock.RUnlock()

	wf, exists := workflows[name]
	if !exists {
		return nil, fmt.Errorf("workflow with name '%s' does not exist", name)
	}

	return wf, nil
}

func ListWorkflows() []string {
	workflowLock.RLock()
	defer workflowLock.RUnlock()

	names := make([]string, 0, len(workflows))
	for name := range workflows {
		names = append(names, name)
	}
	return names
}
