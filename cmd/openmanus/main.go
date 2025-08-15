package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"openmanus-go/internal/agent"
	"openmanus-go/internal/config"
	"openmanus-go/internal/flow"
	"openmanus-go/internal/otel"
	"openmanus-go/internal/tools"
)

func main() {
	root := &cobra.Command{Use: "openmanus", Short: "OpenManus-Go Phase4"}

	var cfg *config.Config
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		c, err := config.Load()
		if err != nil {
			return err
		}
		cfg = c
		// init otel
		tp, err := otel.InitTracer(context.Background(), cfg.OTel.ServiceName, cfg.OTel.Stdout)
		if err != nil {
			log.Error().Err(err).Msg("init otel")
		} else {
			_ = tp
		}
		return nil
	}

	root.AddCommand(planCmd(&cfg))
	root.AddCommand(runCmd(&cfg))
	root.AddCommand(toolsCmd(&cfg))

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildRegistry(cfg *config.Config) *tools.Registry {
	reg := tools.NewRegistry()
	reg.Register(&tools.EchoTool{})
	reg.Register(&tools.HTTPGetTool{Timeout: time.Duration(cfg.OpenAI.TimeoutSeconds) * time.Second})
	reg.Register(&tools.FileReadTool{BaseDir: "."})
	return reg
}

func planCmd(cfg **config.Config) *cobra.Command {
	var fromStdin bool
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Ask planner to produce and execute steps (planner loop) and print plan/result",
		RunE: func(cmd *cobra.Command, args []string) error {
			var prompt string
			if fromStdin {
				data, _ := os.ReadFile("/dev/stdin")
				prompt = string(data)
			} else {
				if len(args) == 0 {
					return fmt.Errorf("provide prompt or use --stdin")
				}
				prompt = strings.Join(args, " ")
			}
			reg := buildRegistry(*cfg)
			p := agent.NewPlanner(*cfg, reg)
			steps, result, err := p.RunPlanLoop(context.Background(), prompt, 6)
			if err != nil {
				return err
			}
			b, _ := json.MarshalIndent(steps, "", "  ")
			fmt.Println("PLAN STEPS:")
			fmt.Println(string(b))
			fmt.Println("RESULT:", result)
			return nil
		},
	}
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "read prompt from stdin")
	return cmd
}

func runCmd(cfg **config.Config) *cobra.Command {
	var fromStdin bool
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Plan + run (planner loop then flow runner for produced steps)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var prompt string
			if fromStdin {
				data, _ := os.ReadFile("/dev/stdin")
				prompt = string(data)
			} else {
				if len(args) == 0 {
					return fmt.Errorf("provide prompt or use --stdin")
				}
				prompt = strings.Join(args, " ")
			}
			reg := buildRegistry(*cfg)
			p := agent.NewPlanner(*cfg, reg)
			steps, result, err := p.RunPlanLoop(context.Background(), prompt, 8)
			if err != nil {
				return err
			}
			flowSteps := []flow.Step{}
			for _, s := range steps {
				fs := flow.Step{Kind: s.Kind, Name: s.Name, Input: s.Input}
				flowSteps = append(flowSteps, fs)
			}
			runner := flow.NewRunner(reg)
			res, err := runner.Run(context.Background(), flowSteps)
			if err != nil {
				return err
			}
			rb, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println("FLOW RESULTS:")
			fmt.Println(string(rb))
			fmt.Println("FINAL RESULT:", result)
			return nil
		},
	}
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "read prompt from stdin")
	return cmd
}

func toolsCmd(cfg **config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "tools",
		Short: "List tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := buildRegistry(*cfg)
			for _, t := range reg.List() {
				fmt.Printf("- %s: %s\n", t.Name(), t.Desc())
			}
			return nil
		},
	}
}
