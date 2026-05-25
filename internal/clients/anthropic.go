// Package clients · Anthropic — minimal Messages API client.
//
// We avoid the official SDK to keep deps lean — the JSON contract is
// stable and small. Supports non-streaming completion (used here) and
// SSE streaming (returns a channel).
package clients

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Anthropic struct {
	APIKey  string
	Model   string
	HTTP    *http.Client
	Version string
}

func NewAnthropic(apiKey, model string) *Anthropic {
	return &Anthropic{
		APIKey:  apiKey,
		Model:   model,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
		Version: "2023-06-01",
	}
}

type Message struct {
	Role    string `json:"role"`    // 'user' | 'assistant'
	Content string `json:"content"`
}

type messageReq struct {
	Model     string    `json:"model"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
	Stream    bool      `json:"stream,omitempty"`
}

type messageRes struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Complete returns the assistant's response text in a single round-trip.
func (a *Anthropic) Complete(ctx context.Context, system string, messages []Message, maxTokens int) (string, error) {
	if a.APIKey == "" {
		return "", errors.New("ANTHROPIC_API_KEY not set")
	}
	body, _ := json.Marshal(messageReq{
		Model:     a.Model,
		System:    system,
		Messages:  messages,
		MaxTokens: maxTokens,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", a.Version)

	res, err := a.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic call: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		buf, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("anthropic %d: %s", res.StatusCode, string(buf))
	}

	var out messageRes
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	var sb strings.Builder
	for _, c := range out.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return sb.String(), nil
}

/* ─── Vision (image input) ──────────────────────────────── */

type contentBlock struct {
	Type   string         `json:"type"`
	Text   string         `json:"text,omitempty"`
	Source *imageSource   `json:"source,omitempty"`
}
type imageSource struct {
	Type      string `json:"type"`       // 'base64'
	MediaType string `json:"media_type"` // 'image/jpeg' etc.
	Data      string `json:"data"`
}

type visionReq struct {
	Model     string `json:"model"`
	System    string `json:"system,omitempty"`
	Messages  []visionMessage `json:"messages"`
	MaxTokens int    `json:"max_tokens"`
}
type visionMessage struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

// CompleteWithImage — single-turn vision request. base64Image must be the
// raw base64 string (no data: URL prefix).
func (a *Anthropic) CompleteWithImage(ctx context.Context, system, prompt, base64Image, mediaType string, maxTokens int) (string, error) {
	if a.APIKey == "" { return "", errors.New("ANTHROPIC_API_KEY not set") }
	body, _ := json.Marshal(visionReq{
		Model: a.Model, System: system, MaxTokens: maxTokens,
		Messages: []visionMessage{
			{Role: "user", Content: []contentBlock{
				{Type: "image", Source: &imageSource{Type: "base64", MediaType: mediaType, Data: base64Image}},
				{Type: "text", Text: prompt},
			}},
		},
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", a.Version)

	res, err := a.HTTP.Do(req)
	if err != nil { return "", err }
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		buf, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("anthropic vision %d: %s", res.StatusCode, string(buf))
	}
	var out messageRes
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil { return "", err }
	var sb strings.Builder
	for _, c := range out.Content { if c.Type == "text" { sb.WriteString(c.Text) } }
	return sb.String(), nil
}

// Stream returns a channel of text deltas, closed on completion or error.
// Caller must drain it. Useful for `/ai/ask` SSE pass-through.
func (a *Anthropic) Stream(ctx context.Context, system string, messages []Message, maxTokens int) (<-chan string, <-chan error) {
	out := make(chan string, 16)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)

		body, _ := json.Marshal(messageReq{
			Model: a.Model, System: system, Messages: messages,
			MaxTokens: maxTokens, Stream: true,
		})
		req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", a.APIKey)
		req.Header.Set("anthropic-version", a.Version)

		res, err := a.HTTP.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			buf, _ := io.ReadAll(res.Body)
			errCh <- fmt.Errorf("anthropic %d: %s", res.StatusCode, string(buf))
			return
		}

		scanner := bufio.NewScanner(res.Body)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}
			var ev struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if json.Unmarshal([]byte(data), &ev) == nil &&
				ev.Type == "content_block_delta" && ev.Delta.Type == "text_delta" {
				select {
				case <-ctx.Done(): return
				case out <- ev.Delta.Text:
				}
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		}
	}()

	return out, errCh
}
