package simplykb

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/LyleLiu666/simplykb/internal/testdb"
)

func BenchmarkHashEmbedderMediumDocument(b *testing.B) {
	ctx := context.Background()
	embedder := NewHashEmbedder(256)
	texts := []string{benchmarkDocument()}

	b.ReportAllocs()
	b.SetBytes(int64(len(texts[0])))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := embedder.Embed(ctx, texts); err != nil {
			b.Fatalf("Embed() error = %v", err)
		}
	}
}

func BenchmarkDefaultSplitterMediumDocument(b *testing.B) {
	splitter := NewDefaultSplitter()
	text := benchmarkDocument()

	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := splitter.Split(text); err != nil {
			b.Fatalf("Split() error = %v", err)
		}
	}
}

func BenchmarkIntegrationUpsertDocument(b *testing.B) {
	ctx := context.Background()
	databaseURL := testdb.DatabaseURL(b)
	schema := testdb.CreateSchema(b, databaseURL, "simplykb_bench")
	store := newBenchmarkStore(b, databaseURL, schema, Config{
		DefaultCollection:   "benchmark",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
	})

	if err := store.Migrate(ctx); err != nil {
		b.Fatalf("Migrate() error = %v", err)
	}

	content := benchmarkDocument()
	b.ReportAllocs()
	b.SetBytes(int64(len(content)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
			DocumentID: fmt.Sprintf("bench-upsert-%d", i),
			Title:      "Benchmark document",
			Content:    content,
			Metadata: map[string]any{
				"benchmark": "upsert",
			},
		}); err != nil {
			b.Fatalf("UpsertDocument() error = %v", err)
		}
	}
}

func BenchmarkIntegrationUpsertDocumentNoop(b *testing.B) {
	ctx := context.Background()
	databaseURL := testdb.DatabaseURL(b)
	schema := testdb.CreateSchema(b, databaseURL, "simplykb_bench")
	store := newBenchmarkStore(b, databaseURL, schema, Config{
		DefaultCollection:   "benchmark",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
	})

	if err := store.Migrate(ctx); err != nil {
		b.Fatalf("Migrate() error = %v", err)
	}

	req := UpsertDocumentRequest{
		DocumentID: "bench-upsert-noop",
		Title:      "Benchmark document",
		Content:    benchmarkDocument(),
		Metadata: map[string]any{
			"benchmark": "upsert-noop",
		},
	}
	if _, err := store.UpsertDocument(ctx, req); err != nil {
		b.Fatalf("seed UpsertDocument() error = %v", err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(req.Content)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := store.UpsertDocument(ctx, req); err != nil {
			b.Fatalf("UpsertDocument() error = %v", err)
		}
	}
}

func BenchmarkIntegrationSearchHybrid(b *testing.B) {
	ctx := context.Background()
	databaseURL := testdb.DatabaseURL(b)
	schema := testdb.CreateSchema(b, databaseURL, "simplykb_bench")
	store := newBenchmarkStore(b, databaseURL, schema, Config{
		DefaultCollection:   "benchmark",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
	})

	if err := store.Migrate(ctx); err != nil {
		b.Fatalf("Migrate() error = %v", err)
	}

	for i := 0; i < 24; i++ {
		if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
			DocumentID: fmt.Sprintf("bench-search-%02d", i),
			Title:      fmt.Sprintf("Benchmark search %02d", i),
			Content:    benchmarkSearchContent(i),
			Metadata: map[string]any{
				"benchmark": "search",
			},
		}); err != nil {
			b.Fatalf("seed UpsertDocument(%d) error = %v", i, err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hits, err := store.Search(ctx, SearchRequest{
			Query: "hybrid recall benchmark search",
			Limit: 5,
		})
		if err != nil {
			b.Fatalf("Search() error = %v", err)
		}
		if len(hits) == 0 {
			b.Fatal("expected at least one hit")
		}
	}
}

func BenchmarkIntegrationSearchDetailedVectorNoCache(b *testing.B) {
	ctx := context.Background()
	databaseURL := testdb.DatabaseURL(b)
	schema := testdb.CreateSchema(b, databaseURL, "simplykb_bench")
	store := newBenchmarkStore(b, databaseURL, schema, Config{
		DefaultCollection:   "benchmark",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
	})

	seedBenchmarkSearchDocuments(b, ctx, store)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		response, err := store.SearchDetailed(ctx, SearchRequest{
			Query: "vector benchmark query",
			Limit: 5,
			Mode:  SearchModeVector,
		})
		if err != nil {
			b.Fatalf("SearchDetailed() error = %v", err)
		}
		if len(response.Hits) == 0 {
			b.Fatal("expected at least one hit")
		}
	}
}

func BenchmarkIntegrationSearchDetailedVectorWithCache(b *testing.B) {
	ctx := context.Background()
	databaseURL := testdb.DatabaseURL(b)
	schema := testdb.CreateSchema(b, databaseURL, "simplykb_bench")
	store := newBenchmarkStore(b, databaseURL, schema, Config{
		DefaultCollection:       "benchmark",
		EmbeddingDimensions:     256,
		Embedder:                NewHashEmbedder(256),
		QueryEmbeddingCacheSize: 8,
	})

	seedBenchmarkSearchDocuments(b, ctx, store)

	if _, err := store.SearchDetailed(ctx, SearchRequest{
		Query: "vector benchmark query",
		Limit: 5,
		Mode:  SearchModeVector,
	}); err != nil {
		b.Fatalf("warm SearchDetailed() error = %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		response, err := store.SearchDetailed(ctx, SearchRequest{
			Query: "vector benchmark query",
			Limit: 5,
			Mode:  SearchModeVector,
		})
		if err != nil {
			b.Fatalf("SearchDetailed() error = %v", err)
		}
		if len(response.Hits) == 0 {
			b.Fatal("expected at least one hit")
		}
		if response.Diagnostics.QueryEmbeddingCacheStatus != QueryEmbeddingCacheStatusHit {
			b.Fatalf("expected warm benchmark query to report hit status, got %q", response.Diagnostics.QueryEmbeddingCacheStatus)
		}
	}
}

func benchmarkDocument() string {
	paragraphs := []string{
		"simplykb keeps retrieval inside a Go service so a team does not need to stand up a second search product just to evaluate hybrid recall.",
		"ParadeDB supplies BM25 and pgvector in the same database, which keeps migrations and local setup much easier to reason about during early rollout.",
		"Stable chunk identifiers, deterministic upsert flow, and small public APIs help external teams judge upgrade risk with less guesswork.",
		"Benchmarks in this repository are meant to be reproducible baselines for comparison, not marketing claims about maximum throughput on every machine.",
	}
	return strings.Join(paragraphs, "\n\n")
}

func benchmarkSearchContent(i int) string {
	if i%2 == 0 {
		return fmt.Sprintf("Document %d explains hybrid recall, benchmark search, and upgrade confidence for embedded Go services.", i)
	}
	return fmt.Sprintf("Document %d focuses on ParadeDB operations, metadata filters, and reproducible benchmark runs.", i)
}

func seedBenchmarkSearchDocuments(b *testing.B, ctx context.Context, store *Store) {
	b.Helper()

	if err := store.Migrate(ctx); err != nil {
		b.Fatalf("Migrate() error = %v", err)
	}

	for i := 0; i < 24; i++ {
		if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
			DocumentID: fmt.Sprintf("bench-search-%02d", i),
			Title:      fmt.Sprintf("Benchmark search %02d", i),
			Content:    benchmarkSearchContent(i),
			Metadata: map[string]any{
				"benchmark": "search",
			},
		}); err != nil {
			b.Fatalf("seed UpsertDocument(%d) error = %v", i, err)
		}
	}
}

func newBenchmarkStore(b *testing.B, databaseURL string, schema string, cfg Config) *Store {
	b.Helper()

	ctx := context.Background()
	cfg.DatabaseURL = testdb.URLWithSearchPath(b, databaseURL, schema)
	store, err := New(ctx, cfg)
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	b.Cleanup(store.Close)
	return store
}
