
package agent

import (
	"context"
	"fmt"

	"openmanus-go/internal/config"
	"openmanus-go/internal/tools"
)

type Agent struct {
	Cfg      *config.Config
	Tools    *tools.Registry
	Selector ToolSelector
	Planner  Planner
}

type AgentResult struct {
	Mode     string       `json:"mode"`
	ToolName string       `json:"tool_name,omitempty"`
	ToolOut  tools.Output `json:"tool_out,omitempty"`
	LLMOut   string       `json:"llm_out,omitempty"`
}

func New(cfg *config.Config, reg *tools.Registry) *Agent {
	return &Agent{Cfg: cfg, Tools: reg, Selector: RuleBasedSelector{}, Planner: Planner{}}
}

func (a *Agent) Act(ctx context.Context, prompt string) (*AgentResult, error) {
	if name, in, ok := a.Selector.Select(ctx, prompt, a.Tools); ok {
		t, ok := a.Tools.Get(name); if !ok { return nil, fmt.Errorf("selected tool not found: %s", name) }
		out, err := t.Run(ctx, in); if err != nil { return nil, err }
		return &AgentResult{Mode: "tool", ToolName: name, ToolOut: out}, nil
	}
	resp, err := Prompt(ctx, a.Cfg, prompt); if err != nil { return nil, err }
	return &AgentResult{Mode: "llm", LLMOut: resp}, nil
}
