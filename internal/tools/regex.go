
package tools

import (
	"context"
	"regexp"
)

type RegexExtractTool struct{}

func (t *RegexExtractTool) Name() string { return "regex_extract" }
func (t *RegexExtractTool) Desc() string { return "Run a regex on input text and return all matches. Inputs: text, pattern" }
func (t *RegexExtractTool) Schema() Schema { return Schema{Name: t.Name(), Desc: t.Desc(), Inputs: map[string]string{"text":"string","pattern":"string"}} }

func (t *RegexExtractTool) Run(ctx context.Context, in Input) (Output, error) {
	text, _ := in["text"].(string)
	pat, _ := in["pattern"].(string)
	re := regexp.MustCompile(pat)
	m := re.FindAllString(text, -1)
	return Output{"matches": m}, nil
}
