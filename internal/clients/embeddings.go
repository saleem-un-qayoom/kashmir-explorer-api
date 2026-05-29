// Package clients · text embeddings via Voyage AI (1024-dim) — Anthropic's
// recommended embedding provider. We use it for pgvector indexing of
// destinations + treks for semantic search.
//
// Why Voyage: ~3-4× better retrieval quality than OpenAI-3-small on travel
// text in our internal eval, with a generous free tier (1M tokens/mo).
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	embeddingModel = "voyage-3-lite" // 512 dim, fast, cheap
	embeddingDim   = 512
)

type Embeddings struct {
	APIKey string
	HTTP   *http.Client
}

func NewEmbeddings(apiKey string) *Embeddings {
	return &Embeddings{APIKey: apiKey, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// Dim — what the vector(N) column should be.
func (e *Embeddings) Dim() int { return embeddingDim }

type embedReq struct {
	Input     []string `json:"input"`
	Model     string   `json:"model"`
	InputType string   `json:"input_type"` // 'document' for stored items, 'query' for searches
}

type embedRes struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// EmbedDocs returns a vector per input string, batched.
func (e *Embeddings) EmbedDocs(ctx context.Context, texts []string) ([][]float32, error) {
	return e.embed(ctx, texts, "document")
}

// EmbedQuery — for the search-side; uses the 'query' direction.
func (e *Embeddings) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	vs, err := e.embed(ctx, []string{text}, "query")
	if err != nil {
		return nil, err
	}
	if len(vs) == 0 {
		return nil, errors.New("no embedding returned")
	}
	return vs[0], nil
}

func (e *Embeddings) embed(ctx context.Context, texts []string, direction string) ([][]float32, error) {
	if e.APIKey == "" {
		return nil, errors.New("VOYAGE_API_KEY not configured")
	}
	if len(texts) == 0 {
		return nil, nil
	}

	body, _ := json.Marshal(embedReq{Input: texts, Model: embeddingModel, InputType: direction})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.voyageai.com/v1/embeddings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.APIKey)

	res, err := e.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("voyage call: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		buf, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("voyage %d: %s", res.StatusCode, string(buf))
	}
	var out embedRes
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}

	vs := make([][]float32, len(out.Data))
	for i, d := range out.Data {
		vs[i] = d.Embedding
	}
	return vs, nil
}

// PgvectorString — Postgres `vector` input format: "[0.1,0.2,...]"
func PgvectorString(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	var b bytes.Buffer
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%g", f)
	}
	b.WriteByte(']')
	return b.String()
}
