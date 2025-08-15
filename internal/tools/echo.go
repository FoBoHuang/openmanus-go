
package tools

import "context"

type EchoTool struct{}

func (e *EchoTool) Name() string { return "echo" }
func (e *EchoTool) Desc() string { return "Echo back the input text. Fields: text (string)" }
func (e *EchoTool) Schema() Schema { return Schema{Name: e.Name(), Desc: e.Desc(), Inputs: map[string]string{"text":"string"}} }

func (e *EchoTool) Run(ctx context.Context, in Input) (Output, error) {
	text, _ := in["text"].(string)
	return Output{"text": text}, nil
}
