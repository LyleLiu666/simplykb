package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LyleLiu666/simplykb"
	"github.com/LyleLiu666/simplykb/examples/internal/exampleembed"
	"github.com/LyleLiu666/simplykb/internal/testdb"
)

type capturedEmbeddingRequest struct {
	Method        string
	Authorization string
	Request       capturedEmbeddingPayload
	DecodeErr     error
}

type capturedEmbeddingPayload struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format,omitempty"`
}

func TestIntegrationOpenAICompatibleExampleRoundTripsAgainstDatabase(t *testing.T) {
	ctx := context.Background()
	databaseURL := testdb.DatabaseURL(t)
	schema := testdb.CreateSchema(t, databaseURL, "simplykb_example_test")

	requests := make(chan capturedEmbeddingRequest, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured := capturedEmbeddingRequest{
			Method:        r.Method,
			Authorization: r.Header.Get("Authorization"),
		}
		captured.DecodeErr = json.NewDecoder(r.Body).Decode(&captured.Request)
		requests <- captured

		response := map[string]any{
			"data": make([]map[string]any, 0, len(captured.Request.Input)),
		}
		for index, input := range captured.Request.Input {
			response["data"] = append(response["data"].([]map[string]any), map[string]any{
				"index":     index,
				"embedding": integrationEmbeddingVector(input),
			})
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	embedder, err := exampleembed.Config{
		Provider:   exampleembed.ProviderOpenAICompatible,
		URL:        server.URL,
		APIKey:     "secret",
		Model:      "demo-embedding",
		Dimensions: 3,
		Timeout:    5 * time.Second,
	}.NewEmbedder()
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	store, err := simplykb.New(ctx, simplykb.Config{
		DatabaseURL:         testdb.URLWithSearchPath(t, databaseURL, schema),
		DefaultCollection:   "integration-example",
		EmbeddingDimensions: 3,
		Embedder:            embedder,
	})
	if err != nil {
		t.Fatalf("simplykb.New() error = %v", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	for _, req := range []simplykb.UpsertDocumentRequest{
		{
			DocumentID: "doc-sdk",
			Title:      "Embedded SDK",
			Content:    "simplykb stays inside a Go service and avoids another search service hop.",
		},
		{
			DocumentID: "doc-db",
			Title:      "ParadeDB setup",
			Content:    "ParadeDB gives BM25 plus vectors and keeps local database setup predictable.",
		},
	} {
		if _, err := store.UpsertDocument(ctx, req); err != nil {
			t.Fatalf("UpsertDocument(%s) error = %v", req.DocumentID, err)
		}
	}

	hits, err := store.Search(ctx, simplykb.SearchRequest{
		Query: "go service sdk",
		Limit: 3,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected at least one hit")
	}
	if hits[0].DocumentID != "doc-sdk" {
		t.Fatalf("unexpected top hit: %+v", hits[0])
	}

	capturedUpsertOne := readCapturedEmbeddingRequest(t, requests)
	capturedUpsertTwo := readCapturedEmbeddingRequest(t, requests)
	capturedSearch := readCapturedEmbeddingRequest(t, requests)

	for _, captured := range []capturedEmbeddingRequest{capturedUpsertOne, capturedUpsertTwo, capturedSearch} {
		if captured.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", captured.Method)
		}
		if captured.Authorization != "Bearer secret" {
			t.Fatalf("Authorization = %q", captured.Authorization)
		}
		if captured.DecodeErr != nil {
			t.Fatalf("decode request: %v", captured.DecodeErr)
		}
		if captured.Request.Model != "demo-embedding" {
			t.Fatalf("Model = %q", captured.Request.Model)
		}
		if captured.Request.EncodingFormat != "float" {
			t.Fatalf("EncodingFormat = %q", captured.Request.EncodingFormat)
		}
	}
	if len(capturedSearch.Request.Input) != 1 || capturedSearch.Request.Input[0] != "go service sdk" {
		t.Fatalf("unexpected search embedding request: %+v", capturedSearch.Request)
	}
}

func integrationEmbeddingVector(text string) []float32 {
	normalized := strings.ToLower(text)
	switch {
	case strings.Contains(normalized, "sdk") || strings.Contains(normalized, "go service"):
		return []float32{1, 0, 0}
	case strings.Contains(normalized, "paradedb") || strings.Contains(normalized, "database"):
		return []float32{0, 1, 0}
	default:
		return []float32{0, 0, 1}
	}
}

func readCapturedEmbeddingRequest(t *testing.T, requests <-chan capturedEmbeddingRequest) capturedEmbeddingRequest {
	t.Helper()

	select {
	case captured := <-requests:
		return captured
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for embedding request")
		return capturedEmbeddingRequest{}
	}
}
