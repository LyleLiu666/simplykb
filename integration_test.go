package simplykb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIntegrationUpsertAndSearch(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}

	if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-1",
		Title:      "Hybrid recall",
		Content:    "BM25 is precise for exact names. Vector search helps with meaning.",
	}); err != nil {
		t.Fatalf("UpsertDocument() error = %v", err)
	}

	hits, err := store.Search(ctx, SearchRequest{
		Query: "exact names",
		Limit: 3,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected at least one hit")
	}
	if hits[0].DocumentID != "doc-1" {
		t.Fatalf("unexpected top hit: %+v", hits[0])
	}
	if hits[0].ChunkID == "" {
		t.Fatal("expected stable chunk id")
	}
	if hits[0].ChunkID != "doc-1:000000" {
		t.Fatalf("unexpected chunk id: %s", hits[0].ChunkID)
	}
}

func TestIntegrationMigrateRejectsEmbeddingDimensionDrift(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)

	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
	})
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("initial Migrate() error = %v", err)
	}

	driftedStore := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 128,
		Embedder:            NewHashEmbedder(128),
	})
	err := driftedStore.Migrate(ctx)
	if err == nil {
		t.Fatal("expected dimension drift to fail migration")
	}
	if !strings.Contains(err.Error(), "embedding dimension") {
		t.Fatalf("expected embedding dimension error, got %v", err)
	}
}

func TestIntegrationUpsertRejectsEmptySplitterOutput(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
		Splitter:            emptySplitter{},
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	_, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-1",
		Title:      "Empty chunks",
		Content:    "this should not be silently accepted",
	})
	if err == nil {
		t.Fatal("expected empty splitter output to be rejected")
	}
	if !strings.Contains(err.Error(), "no chunks") {
		t.Fatalf("expected no chunks error, got %v", err)
	}
}

func TestIntegrationDeleteDocumentTrimsDocumentID(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-1",
		Title:      "Delete me",
		Content:    "trimmed delete should remove this document",
	}); err != nil {
		t.Fatalf("UpsertDocument() error = %v", err)
	}

	if err := store.DeleteDocument(ctx, "", " doc-1 "); err != nil {
		t.Fatalf("DeleteDocument() error = %v", err)
	}

	var count int
	if err := store.pool.QueryRow(ctx, `
SELECT count(*)
FROM kb_documents
WHERE collection = $1 AND external_id = $2
`, "integration", "doc-1").Scan(&count); err != nil {
		t.Fatalf("count documents: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected document to be deleted, found %d rows", count)
	}
}

func TestIntegrationMigrateDropsRedundantIndexes(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	var count int
	if err := store.pool.QueryRow(ctx, `
SELECT count(*)
FROM pg_indexes
WHERE schemaname = current_schema()
  AND indexname = ANY($1)
`, []string{
		"kb_documents_collection_idx",
		"kb_chunks_collection_idx",
		"kb_chunks_key_idx",
	}).Scan(&count); err != nil {
		t.Fatalf("count redundant indexes: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected redundant indexes to be removed, found %d", count)
	}
}

type emptySplitter struct{}

func (emptySplitter) Split(text string) ([]ChunkDraft, error) {
	return nil, nil
}

func integrationDatabaseURL(t *testing.T) string {
	t.Helper()

	databaseURL := os.Getenv("SIMPLYKB_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SIMPLYKB_DATABASE_URL is not set")
	}
	return databaseURL
}

func createIntegrationSchema(t *testing.T, databaseURL string) string {
	t.Helper()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect admin pool: %v", err)
	}
	defer pool.Close()

	schema := fmt.Sprintf("simplykb_test_%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema %s: %v", schema, err)
	}

	t.Cleanup(func() {
		cleanupPool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			t.Fatalf("connect cleanup pool: %v", err)
		}
		defer cleanupPool.Close()
		if _, err := cleanupPool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Fatalf("drop schema %s: %v", schema, err)
		}
	})

	return schema
}

func newIntegrationStore(t *testing.T, databaseURL string, schema string, cfg Config) *Store {
	t.Helper()

	ctx := context.Background()
	cfg.DatabaseURL = databaseURLWithSearchPath(t, databaseURL, schema)
	store, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(store.Close)
	return store
}

func databaseURLWithSearchPath(t *testing.T, databaseURL string, schema string) string {
	t.Helper()

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse database url: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schema+",public")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
