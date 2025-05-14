package registry

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/alan-mat/awe/internal/engine"
)

var (
	executorLock sync.RWMutex
	executors    = make(map[string]engine.Executor)
)

func RegisterExecutor(name string, exec engine.Executor) error {
	executorLock.Lock()
	defer executorLock.Unlock()

	if _, exists := executors[name]; exists {
		return fmt.Errorf("failed to register, executor with name '%s' already exists", name)
	}
	slog.Info("registering executor", "name", name)
	executors[name] = exec
	return nil
}

func GetExecutor(name string) (engine.Executor, error) {
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
