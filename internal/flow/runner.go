package flow

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"openmanus-go/internal/tools"
)

type Step struct {
	Kind   string      `json:"kind"`
	Name   string      `json:"name,omitempty"`
	Input  tools.Input `json:"input,omitempty"`
	Prompt string      `json:"prompt,omitempty"`
}

type Result struct {
	Step   Step `json:"step"`
	Output any  `json:"output"`
}

type Runner struct {
	Tools  *tools.Registry
	Tracer trace.Tracer
}

func NewRunner(reg *tools.Registry) *Runner {
	return &Runner{Tools: reg, Tracer: otel.Tracer("flow.runner")}
}

// resolveTemplates replaces string values containing {{...}} using the context of previous results.
func resolveTemplates(input tools.Input, ctx map[string]any) (tools.Input, error) {
	out := tools.Input{}
	for k, v := range input {
		switch val := v.(type) {
		case string:
			if strings.Contains(val, "{{") {
				tmpl, err := template.New("t").Parse(val)
				if err != nil {
					return nil, err
				}
				var sb strings.Builder
				if err := tmpl.Execute(&sb, ctx); err != nil {
					return nil, err
				}
				out[k] = sb.String()
			} else {
				out[k] = val
			}
		default:
			out[k] = val
		}
	}
	return out, nil
}

func (r *Runner) Run(ctx context.Context, steps []Step) ([]Result, error) {
	tracer := r.Tracer
	ctx, span := tracer.Start(ctx, "flow.run")
	defer span.End()

	results := make([]Result, 0, len(steps))
	tplCtx := map[string]any{"steps": results, "last": nil}

	for i, s := range steps {
		stepSpanCtx, stepSpan := tracer.Start(ctx, fmt.Sprintf("step.%d.%s", i, s.Kind))
		stepSpan.SetAttributes(attribute.String("kind", s.Kind), attribute.String("name", s.Name))

		resolvedInput := s.Input
		if resolvedInput == nil {
			resolvedInput = tools.Input{}
		} else {
			ri, err := resolveTemplates(s.Input, tplCtx)
			if err != nil {
				stepSpan.SetAttributes(attribute.String("error", err.Error()))
				stepSpan.End()
				return nil, fmt.Errorf("template resolve error: %w", err)
			}
			resolvedInput = ri
		}

		var out any
		if s.Kind == "tool" {
			t, ok := r.Tools.Get(s.Name)
			if !ok {
				stepSpan.SetAttributes(attribute.String("error", "tool not found"))
				stepSpan.End()
				return nil, fmt.Errorf("tool not found: %s", s.Name)
			}
			res, err := t.Run(stepSpanCtx, resolvedInput)
			if err != nil {
				stepSpan.SetAttributes(attribute.String("error", err.Error()))
				stepSpan.End()
				return nil, err
			}
			out = res
		} else if s.Kind == "llm" {
			// In this runner we don't call LLM directly for llm-kind; planner handles it.
			out = map[string]any{"text": s.Prompt}
		} else {
			stepSpan.SetAttributes(attribute.String("note", "unknown kind, skipping"))
			out = map[string]any{"note": "skipped"}
		}

		res := Result{Step: s, Output: out}
		results = append(results, res)

		// prepare tplCtx: convert results to maps so templates can access
		stepsForTpl := make([]map[string]any, 0, len(results))
		for _, r := range results {
			stepsForTpl = append(stepsForTpl, map[string]any{
				"step":   r.Step,
				"output": r.Output,
			})
		}
		tplCtx["steps"] = stepsForTpl
		tplCtx["last"] = out

		stepSpan.End()
	}
	return results, nil
}
