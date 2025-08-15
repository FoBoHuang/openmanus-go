
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"openmanus-go/internal/agent"
	"openmanus-go/internal/bus"
	"openmanus-go/internal/config"
	"openmanus-go/internal/flow"
	"openmanus-go/internal/server"
	"openmanus-go/internal/store"
	"openmanus-go/internal/tools"
)

func main() {
	root := &cobra.Command{ Use: "openmanus", Short: "OpenManus-Go CLI" }

	var cfg *config.Config
	var st *store.Store
	var eb *bus.Bus

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		c, err := config.Load(); if err != nil { return err }
		cfg = c
		eb = bus.New()
		st, err = store.Open(cfg.Persistence.Path)
		return err
	}
	defer func(){ if st != nil { _ = st.Close() } }()

	root.AddCommand(runCmd(&cfg, &st, &eb))
	root.AddCommand(flowCmd(&cfg, &st, &eb))
	root.AddCommand(planCmd(&cfg, &st, &eb))
	root.AddCommand(serveCmd(&cfg, &st, &eb))
	root.AddCommand(toolsCmd(&cfg))

	if err := root.Execute(); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}

func buildAgent(cfg *config.Config) *agent.Agent {
	reg := tools.NewRegistry()
	reg.Register(&tools.EchoTool{})
	reg.Register(&tools.HTTPGetTool{Timeout: time.Duration(cfg.OpenAI.TimeoutSeconds) * time.Second})
	reg.Register(&tools.FileReadTool{BaseDir: "."})
	reg.Register(&tools.RegexExtractTool{})
	reg.Register(&tools.JSONPathTool{})
	return agent.New(cfg, reg)
}

func readPromptOrStdin(prompt string, fromStdin bool) (string, error) {
	var text string
	if fromStdin {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 { return "", fmt.Errorf("--stdin specified but no data piped") }
		reader := bufio.NewReader(os.Stdin)
		sb := strings.Builder{}
		for {
			line, err := reader.ReadString('\n')
			sb.WriteString(line)
			if err != nil { break }
		}
		text = strings.TrimSpace(sb.String())
	} else {
		text = prompt
	}
	if strings.TrimSpace(text) == "" { return "", fmt.Errorf("empty prompt, use --prompt or --stdin") }
	return text, nil
}

func runCmd(cfg **config.Config, st **store.Store, eb **bus.Bus) *cobra.Command {
	var prompt string
	var fromStdin bool
	cmd := &cobra.Command{ Use: "run", Short: "Agent plan+execute (autonomous)",
		RunE: func(cmd *cobra.Command, args []string) error {
			text, err := readPromptOrStdin(prompt, fromStdin); if err != nil { return err }
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration((*cfg).OpenAI.TimeoutSeconds+5)*time.Second); defer cancel()
			a := buildAgent(*cfg)
			r := flow.NewRunner(a, *eb, *st)
			out, err := r.PlanAndRun(ctx, text); if err != nil { return err }
			b, _ := json.MarshalIndent(out, "", "  "); fmt.Println(string(b))
			return nil
		},
	}
	cmd.Flags().StringVar(&prompt, "prompt", "", "Goal/Prompt text")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "Read prompt from stdin")
	return cmd
}

func planCmd(cfg **config.Config, st **store.Store, eb **bus.Bus) *cobra.Command {
	var prompt string
	var fromStdin bool
	cmd := &cobra.Command{ Use: "plan", Short: "LLM plan only (prints steps)",
		RunE: func(cmd *cobra.Command, args []string) error {
			text, err := readPromptOrStdin(prompt, fromStdin); if err != nil { return err }
			a := buildAgent(*cfg)
			steps, err := a.Planner.Plan(context.Background(), *cfg, a.Tools, text); if err != nil { return err }
			b, _ := json.MarshalIndent(steps, "", "  "); fmt.Println(string(b))
			return nil
		},
	}
	cmd.Flags().StringVar(&prompt, "prompt", "", "Goal text")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "Read goal from stdin")
	return cmd
}

func flowCmd(cfg **config.Config, st **store.Store, eb **bus.Bus) *cobra.Command {
	var stepsJSON string
	cmd := &cobra.Command{ Use: "flow", Short: "Run a flow via --steps JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			if stepsJSON == "" { return fmt.Errorf("provide --steps JSON") }
			var steps []flow.Step
			if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil { return err }
			a := buildAgent(*cfg)
			r := flow.NewRunner(a, *eb, *st)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration((*cfg).OpenAI.TimeoutSeconds+5)*time.Second); defer cancel()
			results, err := r.Run(ctx, steps); if err != nil { return err }
			b, _ := json.MarshalIndent(results, "", "  "); fmt.Println(string(b))
			return nil
		},
	}
	cmd.Flags().StringVar(&stepsJSON, "steps", "", "JSON array of steps")
	return cmd
}

func serveCmd(cfg **config.Config, st **store.Store, eb **bus.Bus) *cobra.Command {
	var port int
	cmd := &cobra.Command{ Use: "serve", Short: "Start HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			a := buildAgent(*cfg)
			srv := server.New(port, a, *eb, *st, (*cfg).Observability.EnableMetrics, (*cfg).Observability.EnablePProf)
			return srv.Start((*cfg).Observability.EnableMetrics, (*cfg).Observability.EnablePProf)
		},
	}
	cmd.Flags().IntVar(&port, "port", 9000, "port to listen on")
	return cmd
}

func toolsCmd(cfg **config.Config) *cobra.Command {
	return &cobra.Command{ Use: "tools", Short: "List tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			a := buildAgent(*cfg)
			ts := a.Tools.List()
			for _, t := range ts { fmt.Printf("- %s: %s\n", t.Name(), t.Desc()) }
			return nil
		},
	}
}
