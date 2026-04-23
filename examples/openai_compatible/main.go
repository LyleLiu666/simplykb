package main

import (
	"context"
	"fmt"
	"log"

	"github.com/LyleLiu666/simplykb"
	"github.com/LyleLiu666/simplykb/examples/internal/exampleembed"
	"github.com/LyleLiu666/simplykb/examples/internal/exampleenv"
)

type exampleConfig struct {
	DatabaseURL    string
	Collection     string
	EmbedderConfig exampleembed.Config
}

func main() {
	ctx := context.Background()

	cfg, err := loadExampleConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	embedder, err := cfg.EmbedderConfig.NewEmbedder()
	if err != nil {
		log.Fatalf("create embedder: %v", err)
	}

	store, err := simplykb.New(ctx, simplykb.Config{
		DatabaseURL:         cfg.DatabaseURL,
		DefaultCollection:   cfg.Collection,
		EmbeddingDimensions: cfg.EmbedderConfig.Dimensions,
		Embedder:            embedder,
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
	embedderConfig, err := exampleembed.LoadOpenAICompatibleConfigFromEnv()
	if err != nil {
		return exampleConfig{}, err
	}

	return exampleConfig{
		DatabaseURL:    defaultDatabaseURL(),
		Collection:     exampleenv.StringOrDefault("SIMPLYKB_COLLECTION", "demo"),
		EmbedderConfig: embedderConfig,
	}, nil
}

func defaultDatabaseURL() string {
	return exampleenv.DefaultDatabaseURL()
}
