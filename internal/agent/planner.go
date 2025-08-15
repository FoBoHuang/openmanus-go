
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"openmanus-go/internal/config"
	"openmanus-go/internal/flow"
	"openmanus-go/internal/tools"
)

type Planner struct{}

type planStep struct {
	Kind   string         `json:"kind"`
	Name   string         `json:"name,omitempty"`
	Input  map[string]any `json:"input,omitempty"`
	Prompt string         `json:"prompt,omitempty"`
}

func (p Planner) Plan(ctx context.Context, cfg *config.Config, reg *tools.Registry, goal string) ([]flow.Step, error) {
	// Compact tool schema description
	sb := strings.Builder{}
	for _, s := range reg.Schemas() {
		sb.WriteString(fmt.Sprintf("- %s: %s | inputs: %v\n", s.Name, s.Desc, s.Inputs))
	}
	sys := "You are a planning assistant. Given a user goal and available tools, plan a small sequence of steps as JSON. Use only provided tools. Use at most 5 steps."
	user := fmt.Sprintf("TOOLS:\n%s\nGOAL: %s\nReturn JSON array of steps; each step is one of: {kind:\"tool\", name:\"...\", input:{...}} or {kind:\"llm\", prompt:\"...\"}. No prose.", sb.String(), goal)
	out, err := Chat(ctx, cfg, []ChatMessage{{Role: "system", Content: sys}, {Role: "user", Content: user}})
	if err != nil { return nil, err }
	dec := json.NewDecoder(strings.NewReader(out))
	dec.DisallowUnknownFields()
	var steps []planStep
	if err := dec.Decode(&steps); err != nil {
		// try to extract JSON substring
		start := strings.Index(out, "[")
		end := strings.LastIndex(out, "]")
		if start >= 0 && end > start {
			if err := json.Unmarshal([]byte(out[start:end+1]), &steps); err != nil {
				return nil, fmt.Errorf("planner parse: %w | raw: %s", err, out)
			}
		} else {
			return nil, fmt.Errorf("planner returned non-JSON: %s", out)
		}
	}
	res := make([]flow.Step, 0, len(steps))
	for _, s := range steps { res = append(res, flow.Step{Kind: s.Kind, Name: s.Name, Input: s.Input, Prompt: s.Prompt}) }
	return res, nil
}
