package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/LyleLiu666/simplykb"
)

const (
	defaultLocalUser = "simplykb"
	defaultLocalPass = "simplykb"
	defaultLocalDB   = "simplykb"
	defaultLocalPort = "25432"
)

func main() {
	ctx := context.Background()
	databaseURL := defaultDatabaseURL()

	store, err := simplykb.New(ctx, simplykb.Config{
		DatabaseURL:         databaseURL,
		DefaultCollection:   "demo",
		EmbeddingDimensions: 256,
		Embedder:            simplykb.NewHashEmbedder(256),
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
			DocumentID: "doc-bm25",
			Title:      "BM25 notes",
			Content:    "BM25 is often better than vectors for exact names, short queries, logs, and code symbols.",
			Tags:       []string{"keyword", "ranking"},
		},
		{
			DocumentID: "doc-vector",
			Title:      "Vector notes",
			Content:    "Vector search is useful when the question and the answer use different wording but similar meaning.",
			Tags:       []string{"vector", "semantic"},
		},
		{
			DocumentID: "doc-hybrid",
			Title:      "Hybrid recall",
			Content:    "A small knowledge base often works best with BM25 plus vector recall and a simple fusion step.",
			Tags:       []string{"hybrid"},
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
		Query: "exact names and short queries",
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

func defaultDatabaseURL() string {
	if databaseURL := strings.TrimSpace(os.Getenv("SIMPLYKB_DATABASE_URL")); databaseURL != "" {
		return databaseURL
	}

	user := envOrDefault("POSTGRES_USER", defaultLocalUser)
	password := envOrDefault("POSTGRES_PASSWORD", defaultLocalPass)
	database := envOrDefault("POSTGRES_DB", defaultLocalDB)
	port := envOrDefault("PARADEDB_PORT", defaultLocalPort)

	return fmt.Sprintf(
		"postgres://%s:%s@localhost:%s/%s?sslmode=disable",
		user,
		password,
		port,
		database,
	)
}

func envOrDefault(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
