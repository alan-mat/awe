package orchestration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"text/template"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/provider"
	"github.com/alan-mat/awe/internal/registry"
)

var routeExecutorDescriptor = "orchestration.QueryRoute"

const (
	promptLLMSelector = `You are an AI Query Router. Your primary task is to determine the most suitable processing route for a given user query from a list of predefined routes. Each route is defined by a 'key' and a 'description', where the 'description' specifies the types of queries the route is intended to handle.

**Instructions:**

1. Carefully examine the provided user query.
2. For each route in the provided list, compare its 'description' with the user query.
3. Choose the single route whose 'description' most closely matches the user query's intent and information requirements.
4. Assign a confidence score (a floating-point number between 0.0 and 1.0) to your selection. A score of 1.0 indicates you are completely certain the chosen route is the best match, while 0.0 indicates no confidence.
5. Response using JSON.

**Inputs:**

User Query:
{{.Query}}

Available Routes (JSON):
{{.Routes}}
`
)

func init() {
	exec, err := NewRouteExecutor()
	if err != nil {
		slog.Error("failed to initialize executor", "name", routeExecutorDescriptor, "err", err)
		return
	}

	err = registry.RegisterExecutor(routeExecutorDescriptor, exec)
	if err != nil {
		slog.Error("failed to register executor", "name", routeExecutorDescriptor)
	}
}

type RouteExecutor struct {
	DefaultLM provider.LM

	operators map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error)

	templateLLMSelector template.Template
}

func NewRouteExecutor() (*RouteExecutor, error) {
	lp, err := provider.NewLM(provider.LMTypeGemini)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize default providers: %w", err)
	}

	templ := template.Must(template.New("promptLLMSelector").Parse(promptLLMSelector))
	e := &RouteExecutor{
		DefaultLM:           lp,
		templateLLMSelector: *templ,
	}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error){
		"llm_selector": e.llmSelector,
	}
	return e, nil
}

func (e RouteExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) *executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "llm_selector"
	}
	slog.Info("executing", "name", routeExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return &executor.ExecutorResult{
			Name:     routeExecutorDescriptor,
			Operator: p.Operator,
			Err: executor.ErrOperatorNotFound{
				ExecutorName: routeExecutorDescriptor,
				OperatorName: p.Operator,
			},
			Values: nil,
		}
	}

	if len(p.Routes) == 0 {
		return &executor.ExecutorResult{
			Name:     routeExecutorDescriptor,
			Operator: p.Operator,
			Err: executor.ErrInvalidParams{
				ExecutorName:  routeExecutorDescriptor,
				MissingParams: []string{"routes"},
			},
			Values: nil,
		}
	}

	vals, err := opFunc(ctx, p)

	return &executor.ExecutorResult{
		Name:     routeExecutorDescriptor,
		Operator: p.Operator,
		Err:      err,
		Values:   vals,
	}
}

func (e RouteExecutor) llmSelector(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	schema := &api.Schema{
		Title: "Chosen route.",
		Type:  api.TypeObject,
		Properties: map[string]*api.Schema{
			"key": {
				Type:        api.TypeString,
				Description: "Key of the matched route",
			},
			"confidence_score": {
				Type:        api.TypeNumber,
				Description: "Confidence score in the selected route.",
			},
		},
		Required: []string{"key", "confidence_score"},
	}

	type availableRoutes struct {
		Key         string `json:"key"`
		Description string `json:"description"`
	}
	ar := make([]availableRoutes, 0, len(p.Routes))
	for _, route := range p.Routes {
		ar = append(ar, availableRoutes{
			Key:         route.Key,
			Description: route.Description,
		})
	}
	routes, err := json.Marshal(ar)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt template for query '%s': %w", p.GetQuery(), err)
	}

	type templatePayload struct {
		Query  string
		Routes string
	}
	tp := templatePayload{Query: p.GetQuery(), Routes: string(routes)}

	var buf bytes.Buffer
	err = e.templateLLMSelector.Execute(&buf, tp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt template for query '%s': %w", p.GetQuery(), err)
	}
	parsedPrompt := buf.String()

	req := api.GenerationRequest{
		Prompt:         parsedPrompt,
		ResponseSchema: schema,
		Temperature:    0.2,
	}

	cs, err := e.DefaultLM.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query router results: %w", err)
	}

	type routerResponse struct {
		Key        string  `json:"key"`
		Confidence float32 `json:"confidence_score"`
	}
	var route routerResponse

	resp, err := api.StreamReadAll(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("failed to read completions stream: %w", err)
	}
	err = json.Unmarshal([]byte(resp), &route)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal query router response: %w", err)
	}

	slog.Info("llm selector results", "key", route.Key, "confidence", route.Confidence)

	return map[string]any{
		"route_key": route.Key,
	}, nil
}
