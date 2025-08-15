
package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"openmanus-go/internal/agent"
	"openmanus-go/internal/bus"
	"openmanus-go/internal/store"
	"openmanus-go/internal/tools"
)

type Step struct {
	Kind   string      `json:"kind"`
	Name   string      `json:"name"`
	Input  tools.Input `json:"input,omitempty"`
	Prompt string      `json:"prompt,omitempty"`
}

type Result struct {
	Step   Step `json:"step"`
	Output any  `json:"output"`
}

type Runner struct{ Agent *agent.Agent; Bus *bus.Bus; Store *store.Store }

func NewRunner(a *agent.Agent, b *bus.Bus, s *store.Store) *Runner { return &Runner{Agent: a, Bus: b, Store: s} }

func (r *Runner) Run(ctx context.Context, steps []Step) ([]Result, error) {
	runID := uuid.NewString()
	results := make([]Result, 0, len(steps))
	r.Bus.Publish(bus.Event{Topic: "run.started", Data: map[string]any{"run_id": runID, "steps": steps}})
	for i, s := range steps {
		r.Bus.Publish(bus.Event{Topic: "step.started", Data: map[string]any{"run_id": runID, "index": i, "step": s}})
		var out any; var err error
		switch s.Kind {
		case "tool":
			t, ok := r.Agent.Tools.Get(s.Name); if !ok { err = fmt.Errorf("step %d: tool not found: %s", i, s.Name) } else { out, err = t.Run(ctx, s.Input) }
		case "llm":
			out, err = agent.Prompt(ctx, r.Agent.Cfg, s.Prompt)
		case "auto":
			ar, e := r.Agent.Act(ctx, s.Prompt); out, err = ar, e
		default:
			err = fmt.Errorf("step %d: unknown kind: %s", i, s.Kind)
		}
		if err != nil {
			r.Bus.Publish(bus.Event{Topic: "step.error", Data: map[string]any{"run_id": runID, "index": i, "error": err.Error()}})
			return nil, err
		}
		res := Result{Step: s, Output: out}
		results = append(results, res)
		r.Bus.Publish(bus.Event{Topic: "step.finished", Data: map[string]any{"run_id": runID, "index": i, "output": out}})
	}
	r.Bus.Publish(bus.Event{Topic: "run.finished", Data: map[string]any{"run_id": runID, "results": results}})
	_ = r.Store.PutRun(runID, results)
	_ = r.Store.PutEvent(time.Now().UnixNano(), map[string]any{"run_id": runID, "topic": "run.finished"})
	return results, nil
}

func (r *Runner) PlanAndRun(ctx context.Context, goal string) ([]Result, error) {
	steps, err := r.Agent.Planner.Plan(ctx, r.Agent.Cfg, r.Agent.Tools, goal); if err != nil { return nil, err }
	return r.Run(ctx, steps)
}
