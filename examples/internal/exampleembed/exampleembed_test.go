package exampleembed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type capturedEmbeddingRequest struct {
	Method        string
	Authorization string
	RequestBody   map[string]any
	DecodeErr     error
}

func TestLoadConfigFromEnvDefaultsToHash(t *testing.T) {
	t.Setenv("SIMPLYKB_EMBEDDER_PROVIDER", "")
	t.Setenv("SIMPLYKB_EMBEDDING_DIMENSIONS", "")
	t.Setenv("SIMPLYKB_EMBEDDING_TIMEOUT_SECONDS", "")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}
	if cfg.Provider != ProviderHash {
		t.Fatalf("Provider = %q, want %q", cfg.Provider, ProviderHash)
	}
	if cfg.Dimensions != 256 {
		t.Fatalf("Dimensions = %d, want 256", cfg.Dimensions)
	}
	if got := int(cfg.Timeout.Seconds()); got != 30 {
		t.Fatalf("Timeout = %d seconds, want 30", got)
	}
}

func TestLoadConfigFromEnvRequiresOpenAICompatibleSettings(t *testing.T) {
	t.Setenv("SIMPLYKB_EMBEDDER_PROVIDER", ProviderOpenAICompatible)
	t.Setenv("SIMPLYKB_EMBEDDING_URL", "")
	t.Setenv("SIMPLYKB_EMBEDDING_MODEL", "")
	t.Setenv("SIMPLYKB_EMBEDDING_DIMENSIONS", "")

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatal("expected missing openai-compatible settings to fail")
	}
	if !strings.Contains(err.Error(), "SIMPLYKB_EMBEDDING_URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfigFromEnvAllowsOpenAICompatibleWithoutAPIKey(t *testing.T) {
	t.Setenv("SIMPLYKB_EMBEDDER_PROVIDER", ProviderOpenAICompatible)
	t.Setenv("SIMPLYKB_EMBEDDING_URL", "https://embed.example/v1/embeddings")
	t.Setenv("SIMPLYKB_EMBEDDING_MODEL", "text-embedding")
	t.Setenv("SIMPLYKB_EMBEDDING_DIMENSIONS", "384")
	t.Setenv("SIMPLYKB_EMBEDDING_API_KEY", "")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}
	if cfg.Provider != ProviderOpenAICompatible {
		t.Fatalf("Provider = %q, want %q", cfg.Provider, ProviderOpenAICompatible)
	}
	if cfg.APIKey != "" {
		t.Fatalf("APIKey = %q, want empty", cfg.APIKey)
	}
}

func TestLoadOpenAICompatibleConfigRejectsConflictingProviderOverride(t *testing.T) {
	t.Setenv("SIMPLYKB_EMBEDDER_PROVIDER", ProviderHash)

	_, err := LoadOpenAICompatibleConfigFromEnv()
	if err == nil {
		t.Fatal("expected conflicting provider override to fail")
	}
	if !strings.Contains(err.Error(), "conflicts with the openai_compatible example") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewEmbedderUsesOptionalAuthorizationHeader(t *testing.T) {
	requests := make(chan capturedEmbeddingRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured := capturedEmbeddingRequest{
			Method:        r.Method,
			Authorization: r.Header.Get("Authorization"),
		}
		captured.DecodeErr = json.NewDecoder(r.Body).Decode(&captured.RequestBody)
		requests <- captured
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"index":     0,
					"embedding": []float32{0.1, 0.2},
				},
			},
		})
	}))
	defer server.Close()

	cfg := Config{
		Provider:   ProviderOpenAICompatible,
		URL:        server.URL,
		Model:      "text-embedding",
		Dimensions: 2,
		Timeout:    5 * time.Second,
	}

	embedder, err := cfg.NewEmbedder()
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	got, err := embedder.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(got) != 1 || len(got[0]) != 2 {
		t.Fatalf("unexpected vectors: %#v", got)
	}

	request := <-requests
	if request.Method != http.MethodPost {
		t.Fatalf("Method = %q, want POST", request.Method)
	}
	if request.Authorization != "" {
		t.Fatalf("Authorization = %q, want empty when API key is unset", request.Authorization)
	}
	if request.DecodeErr != nil {
		t.Fatalf("decode request: %v", request.DecodeErr)
	}
	if request.RequestBody["model"] != "text-embedding" {
		t.Fatalf("model = %#v, want %q", request.RequestBody["model"], "text-embedding")
	}
}

func TestNewEmbedderValidatesDimensionsFromResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"index":     0,
					"embedding": []float32{0.1},
				},
			},
		})
	}))
	defer server.Close()

	cfg := Config{
		Provider:   ProviderOpenAICompatible,
		URL:        server.URL,
		Model:      "text-embedding",
		Dimensions: 2,
		Timeout:    5 * time.Second,
	}

	embedder, err := cfg.NewEmbedder()
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	_, err = embedder.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected dimension mismatch")
	}
	if !strings.Contains(err.Error(), "dimensions mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewEmbedderHandlesLargeSuccessfulResponse(t *testing.T) {
	const (
		vectorCount = 200
		dimensions  = 1536
	)
	response := makeEmbeddingResponse(vectorCount, dimensions)
	responseBody, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if len(responseBody) <= 1<<20 {
		t.Fatalf("response size = %d bytes, want more than 1 MiB", len(responseBody))
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	cfg := Config{
		Provider:   ProviderOpenAICompatible,
		URL:        server.URL,
		Model:      "text-embedding",
		Dimensions: dimensions,
		Timeout:    5 * time.Second,
	}

	embedder, err := cfg.NewEmbedder()
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	texts := make([]string, vectorCount)
	for i := range texts {
		texts[i] = "chunk"
	}

	got, err := embedder.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(got) != vectorCount {
		t.Fatalf("len(got) = %d, want %d", len(got), vectorCount)
	}
	if len(got[0]) != dimensions {
		t.Fatalf("len(got[0]) = %d, want %d", len(got[0]), dimensions)
	}
}

func makeEmbeddingResponse(count int, dimensions int) map[string]any {
	data := make([]map[string]any, 0, count)
	for i := 0; i < count; i++ {
		vector := make([]float32, dimensions)
		for j := range vector {
			vector[j] = float32((i+j)%7) / 10
		}
		data = append(data, map[string]any{
			"index":     i,
			"embedding": vector,
		})
	}
	return map[string]any{"data": data}
}
