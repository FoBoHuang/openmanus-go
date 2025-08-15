package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"openmanus-go/internal/config"
	"openmanus-go/internal/tools"
)

type PlanStep struct {
	Kind  string                 `json:"kind"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
	Note  string                 `json:"note,omitempty"`
}

type Planner struct {
	Cfg    *config.Config
	Tools  *tools.Registry
	Tracer trace.Tracer
}

func NewPlanner(cfg *config.Config, reg *tools.Registry) *Planner {
	return &Planner{Cfg: cfg, Tools: reg, Tracer: otel.Tracer("planner")}
}

// RunPlanLoop: repeatedly ask the LLM for the next action until done or timeout.
func (p *Planner) RunPlanLoop(ctx context.Context, userPrompt string, maxSteps int) ([]PlanStep, string, error) {
	steps := []PlanStep{}
	history := []map[string]any{}
	ctx, span := p.Tracer.Start(ctx, "RunPlanLoop")
	defer span.End()

	for i := 0; i < maxSteps; i++ {
		prompt := buildPlannerPrompt(userPrompt, history)
		span.AddEvent("call-llm", trace.WithAttributes(attribute.Int("step_index", i)))
		res, err := Prompt(ctx, p.Cfg, prompt)
		if err != nil {
			return steps, "", err
		}
		var j any
		if err := json.Unmarshal([]byte(res), &j); err != nil {
			jstr := extractJSON(res)
			if jstr == "" {
				log.Error().Str("raw", res).Msg("planner: failed to parse JSON response")
				// fallback: treat response as final text
				return steps, res, nil
			}
			if err := json.Unmarshal([]byte(jstr), &j); err != nil {
				return steps, "", fmt.Errorf("planner: json unmarshal failed: %w", err)
			}
		}
		// examine j
		if m, ok := j.(map[string]interface{}); ok {
			if done, _ := m["done"].(bool); done {
				result, _ := m["result"].(string)
				return steps, result, nil
			}
			// maybe single step
			if kind, _ := m["kind"].(string); kind != "" {
				ps := PlanStep{Kind: kind}
				if name, _ := m["name"].(string); name != "" {
					ps.Name = name
				}
				if input, _ := m["input"].(map[string]interface{}); input != nil {
					ps.Input = input
				}
				steps = append(steps, ps)
				out, err := p.executeStep(ctx, ps)
				history = append(history, out)
				if err != nil {
					return steps, "", err
				}
				continue
			}
		}
		// array of steps
		if arr, ok := j.([]interface{}); ok && len(arr) > 0 {
			for _, it := range arr {
				m, ok := it.(map[string]interface{})
				if !ok {
					continue
				}
				kind, _ := m["kind"].(string)
				ps := PlanStep{Kind: kind}
				if name, _ := m["name"].(string); name != "" {
					ps.Name = name
				}
				if input, _ := m["input"].(map[string]interface{}); input != nil {
					ps.Input = input
				}
				steps = append(steps, ps)
				out, err := p.executeStep(ctx, ps)
				history = append(history, out)
				if err != nil {
					return steps, "", err
				}
			}
			// continue the loop so LLM can plan next
			continue
		}
		// default: return raw text as result
		return steps, res, nil
	}
	return steps, "", fmt.Errorf("planner: exceeded max steps %d", maxSteps)
}

func buildPlannerPrompt(userPrompt string, history []map[string]any) string {
	b := "You are an autonomous planner. Available tools: echo, http_get, file_read.\n\nUser goal:\n" + userPrompt + "\n\nHistory:\n"
	for i, h := range history {
		b += fmt.Sprintf("Step %d output: %v\n", i, h)
	}
	b += `Please output either {"done": true, "result":"..."} if finished, or a JSON array of steps like [{"kind":"tool","name":"http_get","input":{"url":"https://..."}}, ...] or a single step object. Keep JSON only in the response.`
	return b
}

func extractJSON(s string) string {
	i := -1
	j := -1
	if idx := indexRune(s, '{'); idx >= 0 {
		i = idx
		j = lastIndexRune(s, '}')
	}
	if i == -1 || j == -1 || j < i {
		if idx := indexRune(s, '['); idx >= 0 {
			i = idx
			j = lastIndexRune(s, ']')
		}
	}
	if i >= 0 && j >= 0 && j >= i {
		return s[i : j+1]
	}
	return ""
}

func indexRune(s string, r byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == r {
			return i
		}
	}
	return -1
}

func lastIndexRune(s string, r byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == r {
			return i
		}
	}
	return -1
}

func (p *Planner) executeStep(ctx context.Context, ps PlanStep) (map[string]any, error) {
	ctx, span := p.Tracer.Start(ctx, fmt.Sprintf("planner.step.%s", ps.Kind))
	defer span.End()
	span.SetAttributes(attribute.String("tool", ps.Name))

	if ps.Kind == "tool" {
		t, ok := p.Tools.Get(ps.Name)
		if !ok {
			return map[string]any{"error": "tool not found"}, fmt.Errorf("tool not found: %s", ps.Name)
		}
		out, err := t.Run(ctx, ps.Input)
		if err != nil {
			span.SetAttributes(attribute.String("error", err.Error()))
			return map[string]any{"error": err.Error()}, err
		}
		res := map[string]any{}
		for k, v := range out {
			res[k] = v
		}
		return res, nil
	}

	if ps.Kind == "llm" {
		prompt := ""
		if ps.Input != nil {
			if p0, ok := ps.Input["prompt"].(string); ok {
				prompt = p0
			}
		}
		if prompt == "" {
			return map[string]any{"error": "empty llm prompt"}, fmt.Errorf("empty llm prompt")
		}
		resp, err := Prompt(ctx, p.Cfg, prompt)
		if err != nil {
			return map[string]any{"error": err.Error()}, err
		}
		return map[string]any{"text": resp}, nil
	}

	return map[string]any{}, nil
}
