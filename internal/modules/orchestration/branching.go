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

package orchestration

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/registry"
)

var branchingExecutorDescriptor = "orchestration.Branching"

func init() {
	exec, err := NewBranchingExecutor()
	if err != nil {
		slog.Error("failed to initialize executor", "name", branchingExecutorDescriptor, "err", err)
		return
	}

	err = registry.RegisterExecutor(branchingExecutorDescriptor, exec)
	if err != nil {
		slog.Error("failed to register executor", "name", branchingExecutorDescriptor)
	}
}

type BranchingExecutor struct {
	operators map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error)
}

func NewBranchingExecutor() (*BranchingExecutor, error) {
	e := &BranchingExecutor{}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error){
		"rrf": e.rrf,
	}
	return e, nil
}

func (e BranchingExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) *executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "rrf"
	}
	slog.Info("executing", "name", branchingExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return &executor.ExecutorResult{
			Name:     branchingExecutorDescriptor,
			Operator: p.Operator,
			Err: executor.ErrOperatorNotFound{
				ExecutorName: branchingExecutorDescriptor,
				OperatorName: p.Operator,
			},
			Values: nil,
		}
	}

	if len(p.Branches) == 0 {
		return &executor.ExecutorResult{
			Name:     branchingExecutorDescriptor,
			Operator: p.Operator,
			Err: executor.ErrInvalidParams{
				ExecutorName:  branchingExecutorDescriptor,
				MissingParams: []string{"branches"},
			},
			Values: nil,
		}
	}

	vals, err := opFunc(ctx, p)

	return &executor.ExecutorResult{
		Name:     branchingExecutorDescriptor,
		Operator: p.Operator,
		Err:      err,
		Values:   vals,
	}
}

func (e BranchingExecutor) rrf(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	contextResults := make([][]*api.ScoredDocument, 0, len(p.Branches))
	resultsChan := make(chan map[string]any, len(p.Branches))
	done := make(chan bool)

	go func(resultsChan chan map[string]any) {
		for res := range resultsChan {
			// check result values for context_docs and append if found
			// if there are not context_docs, ignore the results
			if context, ok := res["context_docs"].([]*api.ScoredDocument); ok {
				contextResults = append(contextResults, context)
			}
		}
		done <- true
	}(resultsChan)

	var wg sync.WaitGroup
	for _, branch := range p.Branches {
		wg.Add(1)
		go func(nodes []*executor.WorkflowNode, params *executor.ExecutorParams, resultsChan chan map[string]any) {
			defer wg.Done()
			var result *executor.ExecutorResult

			for _, node := range nodes {
				nodeParams := executor.MakeNodeParams(node, params)
				result = node.Executor.Execute(ctx, nodeParams)

				if result.Err != nil {
					slog.Error("failed to execute node", "error", fmt.Sprintf("(%T): %v", result.Err, result.Err))
					break
				}

				params = executor.ProcessResult(params, result)
			}

			resultsChan <- params.Args
		}(branch.Nodes, p.Copy(), resultsChan)
	}

	wg.Wait()
	close(resultsChan)
	<-done

	if len(contextResults) == 0 {
		// branches didn't generate any context_docs
		// do not perform RRF, simply continue
		return map[string]any{}, nil
	}

	var k float64 = 60
	kArg, err := executor.GetTypedArg[uint64](p, "rrf_k")
	if err == nil {
		k = float64(kArg)
	}
	slog.Info("rrf using", "k", k)

	// RRF algorithm
	rrfResults := make([]*api.ScoredDocument, 0, 10)
	for _, contextList := range contextResults {
		for rank, doc := range contextList {
			rrfScore := 1.0 / (k + float64(rank+1))

			rrfDoc := doc.Copy()
			rrfDoc.Score = rrfScore

			rrfResults = append(rrfResults, rrfDoc)
		}
	}

	// sort resulting scored documents in descending order
	slices.SortFunc(rrfResults, func(a, b *api.ScoredDocument) int {
		if a.Score == b.Score {
			return 0
		}

		less := a.Score > b.Score
		if less {
			return -1
		} else {
			return 1
		}
	})

	limit, err := executor.GetTypedArg[uint64](p, "limit")
	if err == nil {
		rrfResults = rrfResults[:limit]
	}

	return map[string]any{
		"context_docs": rrfResults,
	}, nil
}
