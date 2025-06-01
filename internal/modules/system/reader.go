// Copyright 2025 Alan Matykiewicz
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the
// Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package system

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/registry"
)

var readerExecutorDescriptor = "system.Reader"

func init() {
	e := NewReaderExecutor()
	err := registry.RegisterExecutor(readerExecutorDescriptor, e)
	if err != nil {
		slog.Error("failed to register executor", "name", readerExecutorDescriptor)
	}
}

type ReaderExecutor struct {
	operators map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error)
}

func NewReaderExecutor() *ReaderExecutor {
	e := &ReaderExecutor{}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error){
		"read_dir_base64": e.readDirBase64,
	}
	return e
}

func (e *ReaderExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) *executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "read_dir_base64"
	}
	slog.Info("executing", "name", readerExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return e.buildResult(p.Operator, executor.ErrOperatorNotFound{
			ExecutorName: readerExecutorDescriptor, OperatorName: p.Operator}, nil)
	}

	vals, err := opFunc(ctx, p)

	return e.buildResult(p.Operator, err, vals)
}

func (e *ReaderExecutor) readDirBase64(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	// readDirBase64 requires following parameter args:
	// path - specifies the directory path to read from, may be relative or absolute
	pathArg, err := p.GetArg("path")
	if err != nil {
		return nil, err
	}

	dirPath, ok := pathArg.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'path' must be of type 'string'")
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory '%s': %e", dirPath, err)
	}

	fileContents := make([]*api.FileContent, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(dirPath, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("failed to read file contents, skipping...", "filePath", path)
			continue
		}

		dataBase64 := base64.StdEncoding.EncodeToString(data)
		fileContents = append(fileContents, &api.FileContent{
			Name:    entry.Name(),
			Content: dataBase64,
		})
	}

	if len(fileContents) == 0 {
		return nil, fmt.Errorf("failed to read directory '%s': no files read", dirPath)
	}

	return map[string]any{
		"file_contents": fileContents,
	}, nil
}

func (e *ReaderExecutor) buildResult(operator string, err error, values map[string]any) *executor.ExecutorResult {
	return &executor.ExecutorResult{
		Name:     readerExecutorDescriptor,
		Operator: operator,
		Err:      err,
		Values:   values,
	}
}
