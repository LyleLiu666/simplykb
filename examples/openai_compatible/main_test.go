package main

import (
	"strings"
	"testing"
)

func TestLoadExampleConfigFromEnv(t *testing.T) {
	t.Setenv("SIMPLYKB_DATABASE_URL", "postgres://demo")
	t.Setenv("SIMPLYKB_COLLECTION", "prod")
	t.Setenv("SIMPLYKB_EMBEDDER_PROVIDER", "")
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
	if cfg.EmbedderConfig.Provider != "openai_compatible" {
		t.Fatalf("Provider = %q", cfg.EmbedderConfig.Provider)
	}
	if cfg.EmbedderConfig.Dimensions != 256 {
		t.Fatalf("EmbeddingDimensions = %d", cfg.EmbedderConfig.Dimensions)
	}
	if got := int(cfg.EmbedderConfig.Timeout.Seconds()); got != 12 {
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
