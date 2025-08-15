
package tools

import (
	"context"
	"encoding/json"

	"github.com/PaesslerAG/jsonpath"
)

type JSONPathTool struct{}

func (t *JSONPathTool) Name() string { return "jsonpath" }
func (t *JSONPathTool) Desc() string { return "Evaluate a JSONPath expression on a JSON string. Inputs: json, expr" }
func (t *JSONPathTool) Schema() Schema { return Schema{Name: t.Name(), Desc: t.Desc(), Inputs: map[string]string{"json":"string","expr":"string"}} }

func (t *JSONPathTool) Run(ctx context.Context, in Input) (Output, error) {
	js, _ := in["json"].(string)
	expr, _ := in["expr"].(string)
	var data any
	if err := json.Unmarshal([]byte(js), &data); err != nil { return nil, err }
	val, err := jsonpath.Get(expr, data); if err != nil { return nil, err }
	return Output{"value": val}, nil
}
