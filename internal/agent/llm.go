package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"openmanus-go/internal/config"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func Chat(ctx context.Context, cfg *config.Config, msgs []ChatMessage) (string, error) {
	if cfg.OpenAI.APIKey == "" {
		return "", fmt.Errorf("missing OpenAI API key; set OPENAI_API_KEY")
	}
	reqBody := chatRequest{
		Model:       cfg.OpenAI.Model,
		Messages:    msgs,
		Temperature: cfg.OpenAI.Temperature,
	}
	b, _ := json.Marshal(reqBody)
	url := cfg.OpenAI.BaseURL + "/chat/completions"
	client := &http.Client{Timeout: time.Duration(cfg.OpenAI.TimeoutSeconds) * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAI.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai error: %s", string(body))
	}
	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", err
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("no choices")
	}
	return cr.Choices[0].Message.Content, nil
}

func Prompt(ctx context.Context, cfg *config.Config, prompt string) (string, error) {
	msgs := []ChatMessage{
		{Role: "system", Content: "You are OpenManus-Go, a planning assistant. When asked to produce steps, reply with JSON array of steps with fields: kind (tool|llm|auto), name (tool name), input (object). If finished, reply with {\"done\": true, \"result\": \"...\"}."},
		{Role: "user", Content: prompt},
	}
	return Chat(ctx, cfg, msgs)
}
