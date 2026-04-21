package simplykb

import (
	"strings"
	"testing"
)

func TestConfigValidate(t *testing.T) {
	cfg := Config{
		DatabaseURL:         "postgres://demo",
		EmbeddingDimensions: 64,
		Embedder:            NewHashEmbedder(64),
	}
	if err := cfg.normalized().validate(); err != nil {
		t.Fatalf("validate() error = %v", err)
	}
}

func TestValidateSearchRequest(t *testing.T) {
	store := &Store{
		cfg: Config{
			DefaultCollection:  "default",
			DefaultSearchLimit: 5,
			CandidateLimit:     10,
		}.normalized(),
	}

	req := store.normalizeSearchRequest(SearchRequest{
		Query: "bm25",
	})
	if err := store.validateSearchRequest(req); err != nil {
		t.Fatalf("validateSearchRequest() error = %v", err)
	}
}

func TestValidateSearchRequestRejectsInvalidMetadataFilter(t *testing.T) {
	store := &Store{
		cfg: Config{
			DefaultCollection:  "default",
			DefaultSearchLimit: 5,
			CandidateLimit:     10,
		}.normalized(),
	}

	req := store.normalizeSearchRequest(SearchRequest{
		Query: "bm25",
		MetadataFilter: map[string]any{
			"bad": make(chan int),
		},
	})
	err := store.validateSearchRequest(req)
	if err == nil {
		t.Fatal("expected invalid metadata filter to fail")
	}
}

func TestVectorLiteral(t *testing.T) {
	got := vectorLiteral([]float32{1, 0.25, 0})
	if got != "[1,0.25,0]" {
		t.Fatalf("unexpected vector literal: %s", got)
	}
}

func TestValidateRequiredExtensions(t *testing.T) {
	err := validateRequiredExtensions(map[string]struct{}{
		"vector": {},
	})
	if err == nil {
		t.Fatal("expected missing extension error")
	}
	if !strings.Contains(err.Error(), "plain Postgres") {
		t.Fatalf("expected plain Postgres guidance, got %v", err)
	}
	if !strings.Contains(err.Error(), "pg_search") {
		t.Fatalf("expected missing extension name, got %v", err)
	}
}

func TestValidateRequiredExtensionsAcceptsRequiredSet(t *testing.T) {
	err := validateRequiredExtensions(map[string]struct{}{
		"pg_search": {},
		"vector":    {},
	})
	if err != nil {
		t.Fatalf("expected required extensions to pass, got %v", err)
	}
}
