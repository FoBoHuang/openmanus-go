package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPGetTool struct{ Timeout time.Duration }

func (t *HTTPGetTool) Name() string { return "http_get" }
func (t *HTTPGetTool) Desc() string { return "HTTP GET a URL. Fields: url (string)" }

func (t *HTTPGetTool) Run(ctx context.Context, in Input) (Output, error) {
	url, _ := in["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("missing url")
	}
	client := &http.Client{Timeout: t.Timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return Output{"status": resp.StatusCode, "body": string(body)}, nil
}
