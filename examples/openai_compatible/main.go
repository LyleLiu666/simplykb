package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/LyleLiu666/simplykb"
	"github.com/LyleLiu666/simplykb/examples/internal/exampleenv"
)

type exampleConfig struct {
	DatabaseURL         string
	Collection          string
	EmbeddingURL        string
	EmbeddingAPIKey     string
	EmbeddingModel      string
	EmbeddingDimensions int
	Timeout             time.Duration
}

type openAICompatibleEmbedder struct {
	client             *http.Client
	url                string
	apiKey             string
	model              string
	expectedDimensions int
}

type embeddingsRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format,omitempty"`
}

type embeddingsResponse struct {
	Data []embeddingItem `json:"data"`
}

type embeddingItem struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

func main() {
	ctx := context.Background()

	cfg, err := loadExampleConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	store, err := simplykb.New(ctx, simplykb.Config{
		DatabaseURL:         cfg.DatabaseURL,
		DefaultCollection:   cfg.Collection,
		EmbeddingDimensions: cfg.EmbeddingDimensions,
		Embedder: &openAICompatibleEmbedder{
			client: &http.Client{
				Timeout: cfg.Timeout,
			},
			url:                cfg.EmbeddingURL,
			apiKey:             cfg.EmbeddingAPIKey,
			model:              cfg.EmbeddingModel,
			expectedDimensions: cfg.EmbeddingDimensions,
		},
	})
	if err != nil {
		log.Fatalf("create store: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate schema: %v", err)
	}

	docs := []simplykb.UpsertDocumentRequest{
		{
			DocumentID: "doc-sdk",
			Title:      "Embedded SDK",
			Content:    "simplykb is meant to live inside a Go service instead of behind another HTTP boundary.",
			Tags:       []string{"sdk", "embedding"},
		},
		{
			DocumentID: "doc-paradedb",
			Title:      "ParadeDB dependency",
			Content:    "Docker helps local development by starting the ParadeDB dependency with predictable defaults.",
			Tags:       []string{"paradedb", "docker"},
		},
	}

	for _, doc := range docs {
		stats, err := store.UpsertDocument(ctx, doc)
		if err != nil {
			log.Fatalf("upsert %s: %v", doc.DocumentID, err)
		}
		fmt.Printf("indexed %s with %d chunks\n", stats.DocumentID, stats.ChunkCount)
	}

	hits, err := store.Search(ctx, simplykb.SearchRequest{
		Query: "go service sdk",
		Limit: 3,
	})
	if err != nil {
		log.Fatalf("search: %v", err)
	}

	fmt.Println("\nTop hits:")
	for _, hit := range hits {
		fmt.Printf("- %s chunk=%d score=%.4f keyword=%.4f vector=%.4f\n", hit.DocumentID, hit.ChunkNumber, hit.Score, hit.KeywordScore, hit.VectorScore)
		fmt.Printf("  snippet: %s\n", hit.Snippet)
	}
}

func loadExampleConfig() (exampleConfig, error) {
	dimensions, err := intEnvOrDefault("SIMPLYKB_EMBEDDING_DIMENSIONS", 0)
	if err != nil {
		return exampleConfig{}, err
	}
	timeoutSeconds, err := intEnvOrDefault("SIMPLYKB_EMBEDDING_TIMEOUT_SECONDS", 30)
	if err != nil {
		return exampleConfig{}, err
	}

	cfg := exampleConfig{
		DatabaseURL:         defaultDatabaseURL(),
		Collection:          exampleenv.StringOrDefault("SIMPLYKB_COLLECTION", "demo"),
		EmbeddingURL:        strings.TrimSpace(os.Getenv("SIMPLYKB_EMBEDDING_URL")),
		EmbeddingAPIKey:     strings.TrimSpace(os.Getenv("SIMPLYKB_EMBEDDING_API_KEY")),
		EmbeddingModel:      strings.TrimSpace(os.Getenv("SIMPLYKB_EMBEDDING_MODEL")),
		EmbeddingDimensions: dimensions,
		Timeout:             time.Duration(timeoutSeconds) * time.Second,
	}

	switch {
	case cfg.EmbeddingURL == "":
		return exampleConfig{}, errors.New("SIMPLYKB_EMBEDDING_URL is required")
	case cfg.EmbeddingAPIKey == "":
		return exampleConfig{}, errors.New("SIMPLYKB_EMBEDDING_API_KEY is required")
	case cfg.EmbeddingModel == "":
		return exampleConfig{}, errors.New("SIMPLYKB_EMBEDDING_MODEL is required")
	case cfg.EmbeddingDimensions <= 0:
		return exampleConfig{}, errors.New("SIMPLYKB_EMBEDDING_DIMENSIONS must be greater than 0")
	}

	return cfg, nil
}

func (e *openAICompatibleEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errors.New("embedder is nil")
	}
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	body, err := json.Marshal(embeddingsRequest{
		Model:          e.model,
		Input:          texts,
		EncodingFormat: "float",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build embedding request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embedding api: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read embedding response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding api returned %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	var payload embeddingsResponse
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(payload.Data) != len(texts) {
		return nil, fmt.Errorf("embedding api returned %d vectors for %d texts", len(payload.Data), len(texts))
	}

	sort.Slice(payload.Data, func(i, j int) bool {
		return payload.Data[i].Index < payload.Data[j].Index
	})

	out := make([][]float32, 0, len(payload.Data))
	for i, item := range payload.Data {
		if item.Index != i {
			return nil, fmt.Errorf("embedding api returned unexpected index %d at position %d", item.Index, i)
		}
		if len(item.Embedding) != e.expectedDimensions {
			return nil, fmt.Errorf("embedding dimensions mismatch: got %d want %d", len(item.Embedding), e.expectedDimensions)
		}
		out = append(out, item.Embedding)
	}

	return out, nil
}

func defaultDatabaseURL() string {
	return exampleenv.DefaultDatabaseURL()
}

func intEnvOrDefault(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0", key)
	}
	return value, nil
}
