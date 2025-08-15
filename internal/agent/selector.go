package agent

import (
	"context"
	"regexp"
	"strings"

	"openmanus-go/internal/tools"
)

type ToolSelector interface {
	Select(ctx context.Context, prompt string, reg *tools.Registry) (name string, input tools.Input, ok bool)
}

type RuleBasedSelector struct{}

var urlRe = regexp.MustCompile(`https?://[^\s]+`)

func (r RuleBasedSelector) Select(ctx context.Context, prompt string, reg *tools.Registry) (string, tools.Input, bool) {
	p := strings.ToLower(prompt)

	if strings.Contains(p, "echo:") {
		rest := strings.TrimSpace(strings.SplitN(prompt, "echo:", 2)[1])
		return "echo", tools.Input{"text": rest}, true
	}

	if strings.Contains(p, "http") {
		u := urlRe.FindString(prompt)
		if u != "" {
			return "http_get", tools.Input{"url": u}, true
		}
	}

	if strings.Contains(p, "read file") || strings.Contains(p, "file_read") {
		if strings.Contains(prompt, ":") {
			rel := strings.TrimSpace(strings.SplitN(prompt, ":", 2)[1])
			return "file_read", tools.Input{"path": rel}, true
		}
	}

	return "", nil, false
}
