package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type capturedEmbeddingRequest struct {
	Method        string
	Authorization string
	Request       embeddingsRequest
	DecodeErr     error
}

func TestLoadExampleConfigFromEnv(t *testing.T) {
	t.Setenv("SIMPLYKB_DATABASE_URL", "postgres://demo")
	t.Setenv("SIMPLYKB_COLLECTION", "prod")
	t.Setenv("SIMPLYKB_EMBEDDING_URL", "https://embed.example/v1/embeddings")
	t.Setenv("SIMPLYKB_EMBEDDING_API_KEY", "secret")
	t.Setenv("SIMPLYKB_EMBEDDING_MODEL", "text-embedding")
	t.Setenv("SIMPLYKB_EMBEDDING_DIMENSIONS", "256")
	t.Setenv("SIMPLYKB_EMBEDDING_TIMEOUT_SECONDS", "12")

	cfg, err := loadExampleConfig()
	if err != nil {
		t.Fatalf("loadExampleConfig() error = %v", err)
	}
	if cfg.DatabaseURL != "postgres://demo" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.Collection != "prod" {
		t.Fatalf("Collection = %q", cfg.Collection)
	}
	if cfg.EmbeddingDimensions != 256 {
		t.Fatalf("EmbeddingDimensions = %d", cfg.EmbeddingDimensions)
	}
	if got := int(cfg.Timeout.Seconds()); got != 12 {
		t.Fatalf("Timeout = %d seconds", got)
	}
}

func TestLoadExampleConfigRequiresEmbeddingEnv(t *testing.T) {
	t.Setenv("SIMPLYKB_EMBEDDING_URL", "")
	t.Setenv("SIMPLYKB_EMBEDDING_API_KEY", "")
	t.Setenv("SIMPLYKB_EMBEDDING_MODEL", "")
	t.Setenv("SIMPLYKB_EMBEDDING_DIMENSIONS", "")

	_, err := loadExampleConfig()
	if err == nil {
		t.Fatal("expected missing embedding env to fail")
	}
	if !strings.Contains(err.Error(), "SIMPLYKB_EMBEDDING_URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadExampleConfigRejectsInvalidNumericEnv(t *testing.T) {
	t.Setenv("SIMPLYKB_DATABASE_URL", "postgres://demo")
	t.Setenv("SIMPLYKB_EMBEDDING_URL", "https://embed.example/v1/embeddings")
	t.Setenv("SIMPLYKB_EMBEDDING_API_KEY", "secret")
	t.Setenv("SIMPLYKB_EMBEDDING_MODEL", "text-embedding")
	t.Setenv("SIMPLYKB_EMBEDDING_DIMENSIONS", "abc")
	t.Setenv("SIMPLYKB_EMBEDDING_TIMEOUT_SECONDS", "12")

	_, err := loadExampleConfig()
	if err == nil {
		t.Fatal("expected invalid embedding dimensions to fail")
	}
	if !strings.Contains(err.Error(), "SIMPLYKB_EMBEDDING_DIMENSIONS") {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("SIMPLYKB_EMBEDDING_DIMENSIONS", "256")
	t.Setenv("SIMPLYKB_EMBEDDING_TIMEOUT_SECONDS", "zero")

	_, err = loadExampleConfig()
	if err == nil {
		t.Fatal("expected invalid timeout to fail")
	}
	if !strings.Contains(err.Error(), "SIMPLYKB_EMBEDDING_TIMEOUT_SECONDS") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAICompatibleEmbedderEmbed(t *testing.T) {
	requests := make(chan capturedEmbeddingRequest, 1)
	handlerErrs := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured := capturedEmbeddingRequest{
			Method:        r.Method,
			Authorization: r.Header.Get("Authorization"),
		}
		captured.DecodeErr = json.NewDecoder(r.Body).Decode(&captured.Request)
		requests <- captured

		resp := embeddingsResponse{
			Data: []embeddingItem{
				{Index: 1, Embedding: []float32{0.3, 0.4}},
				{Index: 0, Embedding: []float32{0.1, 0.2}},
			},
		}
		handlerErrs <- json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := &openAICompatibleEmbedder{
		client:             server.Client(),
		url:                server.URL,
		apiKey:             "secret",
		model:              "text-embedding",
		expectedDimensions: 2,
	}

	got, err := embedder.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	captured := <-requests
	if captured.Method != http.MethodPost {
		t.Fatalf("method = %s, want POST", captured.Method)
	}
	if captured.Authorization != "Bearer secret" {
		t.Fatalf("Authorization = %q", captured.Authorization)
	}
	if captured.DecodeErr != nil {
		t.Fatalf("decode request: %v", captured.DecodeErr)
	}
	if captured.Request.Model != "text-embedding" {
		t.Fatalf("Model = %q", captured.Request.Model)
	}
	if err := <-handlerErrs; err != nil {
		t.Fatalf("encode response: %v", err)
	}
	if len(captured.Request.Input) != 2 {
		t.Fatalf("Input length = %d", len(captured.Request.Input))
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d", len(got))
	}
	if got[0][0] != 0.1 || got[1][0] != 0.3 {
		t.Fatalf("unexpected vectors: %#v", got)
	}
}

func TestOpenAICompatibleEmbedderValidatesDimensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embeddingsResponse{
			Data: []embeddingItem{
				{Index: 0, Embedding: []float32{0.1}},
			},
		})
	}))
	defer server.Close()

	embedder := &openAICompatibleEmbedder{
		client:             server.Client(),
		url:                server.URL,
		apiKey:             "secret",
		model:              "text-embedding",
		expectedDimensions: 2,
	}

	_, err := embedder.Embed(context.Background(), []string{"first"})
	if err == nil {
		t.Fatal("expected dimension mismatch")
	}
	if !strings.Contains(err.Error(), "dimensions mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAICompatibleEmbedderHandlesLargeSuccessfulResponse(t *testing.T) {
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

	embedder := &openAICompatibleEmbedder{
		client:             server.Client(),
		url:                server.URL,
		apiKey:             "secret",
		model:              "text-embedding",
		expectedDimensions: dimensions,
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

func makeEmbeddingResponse(count int, dimensions int) embeddingsResponse {
	data := make([]embeddingItem, 0, count)
	for i := 0; i < count; i++ {
		vector := make([]float32, dimensions)
		for j := range vector {
			vector[j] = float32((i+j)%7) / 10
		}
		data = append(data, embeddingItem{
			Index:     i,
			Embedding: vector,
		})
	}
	return embeddingsResponse{Data: data}
}
