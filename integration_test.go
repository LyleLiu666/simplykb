package simplykb

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LyleLiu666/simplykb/internal/testdb"
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

func TestIntegrationSearchFiltersByMetadata(t *testing.T) {
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

	for _, req := range []UpsertDocumentRequest{
		{
			DocumentID: "doc-alpha",
			Title:      "Tenant alpha handbook",
			Content:    "Customer support handbook explains incident response and escalation.",
			Metadata: map[string]any{
				"tenant": "alpha",
				"tier":   "gold",
			},
		},
		{
			DocumentID: "doc-beta",
			Title:      "Tenant beta handbook",
			Content:    "Customer support handbook explains incident response and escalation.",
			Metadata: map[string]any{
				"tenant": "beta",
				"tier":   "silver",
			},
		},
	} {
		if _, err := store.UpsertDocument(ctx, req); err != nil {
			t.Fatalf("UpsertDocument(%s) error = %v", req.DocumentID, err)
		}
	}

	hits, err := store.Search(ctx, SearchRequest{
		Query: "incident response handbook",
		Limit: 5,
		MetadataFilter: map[string]any{
			"tenant": "alpha",
		},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected at least one filtered hit")
	}
	for _, hit := range hits {
		if hit.DocumentID != "doc-alpha" {
			t.Fatalf("expected only doc-alpha hits, got %+v", hit)
		}
	}
}

func TestIntegrationSearchDetailedMatchesSearchAndReportsDiagnostics(t *testing.T) {
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

	for _, req := range []UpsertDocumentRequest{
		{
			DocumentID: "doc-a",
			Title:      "Hybrid recall",
			Content:    "BM25 helps with exact names while vector search helps with meaning.",
		},
		{
			DocumentID: "doc-b",
			Title:      "Vector recall",
			Content:    "Semantic search helps when wording is different but intent is similar.",
		},
	} {
		if _, err := store.UpsertDocument(ctx, req); err != nil {
			t.Fatalf("UpsertDocument(%s) error = %v", req.DocumentID, err)
		}
	}

	searchReq := SearchRequest{
		Query: "exact names with similar meaning",
		Limit: 3,
	}

	hits, err := store.Search(ctx, searchReq)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	deadlineCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	response, err := store.SearchDetailed(deadlineCtx, searchReq)
	if err != nil {
		t.Fatalf("SearchDetailed() error = %v", err)
	}

	assertSearchHitsEqual(t, response.Hits, hits)
	if response.Diagnostics.Mode != SearchModeHybrid {
		t.Fatalf("diagnostics mode = %q, want %q", response.Diagnostics.Mode, SearchModeHybrid)
	}
	if response.Diagnostics.TotalDuration <= 0 {
		t.Fatalf("total duration = %v, want > 0", response.Diagnostics.TotalDuration)
	}
	if response.Diagnostics.KeywordDuration <= 0 {
		t.Fatalf("keyword duration = %v, want > 0", response.Diagnostics.KeywordDuration)
	}
	if response.Diagnostics.VectorDuration <= 0 {
		t.Fatalf("vector duration = %v, want > 0", response.Diagnostics.VectorDuration)
	}
	if response.Diagnostics.KeywordCandidateCount == 0 {
		t.Fatal("expected keyword candidate count to be populated")
	}
	if response.Diagnostics.VectorCandidateCount == 0 {
		t.Fatal("expected vector candidate count to be populated")
	}
	if response.Diagnostics.FusedCandidateCount < len(response.Hits) {
		t.Fatalf("fused candidate count = %d, want >= %d", response.Diagnostics.FusedCandidateCount, len(response.Hits))
	}
	if response.Diagnostics.QueryEmbeddingCacheStatus != QueryEmbeddingCacheStatusDisabled {
		t.Fatalf("cache status = %q, want %q", response.Diagnostics.QueryEmbeddingCacheStatus, QueryEmbeddingCacheStatusDisabled)
	}
	if response.Diagnostics.QueryEmbeddingCacheHit {
		t.Fatal("expected cache hit to be false when cache is disabled")
	}
	if !response.Diagnostics.HadContextDeadline {
		t.Fatal("expected HadContextDeadline to be true")
	}
}

func TestIntegrationSearchDetailedUsesQueryEmbeddingCacheWhenEnabled(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	embedder := &countingEmbedder{dimensions: 256}
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:       "integration",
		EmbeddingDimensions:     256,
		Embedder:                embedder,
		QueryEmbeddingCacheSize: 8,
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-cache",
		Title:      "Cached search",
		Content:    "Vector search helps with meaning and repeated semantic queries.",
	}); err != nil {
		t.Fatalf("UpsertDocument() error = %v", err)
	}

	callsAfterSeed := embedder.CallCount()

	first, err := store.SearchDetailed(ctx, SearchRequest{
		Query: "semantic query",
		Limit: 3,
		Mode:  SearchModeVector,
	})
	if err != nil {
		t.Fatalf("first SearchDetailed() error = %v", err)
	}
	if first.Diagnostics.QueryEmbeddingCacheHit {
		t.Fatal("expected first search to miss cache")
	}
	if first.Diagnostics.QueryEmbeddingCacheStatus != QueryEmbeddingCacheStatusMiss {
		t.Fatalf("first cache status = %q, want %q", first.Diagnostics.QueryEmbeddingCacheStatus, QueryEmbeddingCacheStatusMiss)
	}

	second, err := store.SearchDetailed(ctx, SearchRequest{
		Query: "  semantic query  ",
		Limit: 3,
		Mode:  SearchModeVector,
	})
	if err != nil {
		t.Fatalf("second SearchDetailed() error = %v", err)
	}
	if !second.Diagnostics.QueryEmbeddingCacheHit {
		t.Fatal("expected second search to hit cache")
	}
	if second.Diagnostics.QueryEmbeddingCacheStatus != QueryEmbeddingCacheStatusHit {
		t.Fatalf("second cache status = %q, want %q", second.Diagnostics.QueryEmbeddingCacheStatus, QueryEmbeddingCacheStatusHit)
	}
	if got := embedder.CallCount() - callsAfterSeed; got != 1 {
		t.Fatalf("query embedder calls with cache enabled = %d, want 1", got)
	}
	assertSearchHitsEqual(t, first.Hits, second.Hits)
}

func TestIntegrationSearchDetailedWithCacheDisabledEmbedsEveryTime(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	embedder := &countingEmbedder{dimensions: 256}
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            embedder,
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-no-cache",
		Title:      "No cache search",
		Content:    "Repeated vector queries should re-embed when cache is disabled.",
	}); err != nil {
		t.Fatalf("UpsertDocument() error = %v", err)
	}

	callsAfterSeed := embedder.CallCount()
	for i := 0; i < 2; i++ {
		response, err := store.SearchDetailed(ctx, SearchRequest{
			Query: "repeated query",
			Limit: 3,
			Mode:  SearchModeVector,
		})
		if err != nil {
			t.Fatalf("SearchDetailed(%d) error = %v", i, err)
		}
		if response.Diagnostics.QueryEmbeddingCacheHit {
			t.Fatalf("SearchDetailed(%d) unexpectedly reported cache hit", i)
		}
		if response.Diagnostics.QueryEmbeddingCacheStatus != QueryEmbeddingCacheStatusDisabled {
			t.Fatalf("SearchDetailed(%d) cache status = %q, want %q", i, response.Diagnostics.QueryEmbeddingCacheStatus, QueryEmbeddingCacheStatusDisabled)
		}
	}
	if got := embedder.CallCount() - callsAfterSeed; got != 2 {
		t.Fatalf("query embedder calls with cache disabled = %d, want 2", got)
	}
}

func TestIntegrationSearchDetailedAllowsEmbedderToBypassQueryEmbeddingCache(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	embedder := &countingEmbedder{dimensions: 256}
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:       "integration",
		EmbeddingDimensions:     256,
		Embedder:                embedder,
		QueryEmbeddingCacheSize: 8,
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-bypass-cache",
		Title:      "Bypassed cache search",
		Content:    "Some requests should skip the query embedding cache on purpose.",
	}); err != nil {
		t.Fatalf("UpsertDocument() error = %v", err)
	}

	callsAfterSeed := embedder.CallCount()
	bypassCtx := context.WithValue(ctx, integrationQueryCacheBypassKey{}, true)

	for i := 0; i < 2; i++ {
		response, err := store.SearchDetailed(bypassCtx, SearchRequest{
			Query: "bypass cache",
			Limit: 3,
			Mode:  SearchModeVector,
		})
		if err != nil {
			t.Fatalf("SearchDetailed(%d) error = %v", i, err)
		}
		if len(response.Hits) == 0 {
			t.Fatalf("SearchDetailed(%d) expected hits", i)
		}
		if response.Diagnostics.QueryEmbeddingCacheStatus != QueryEmbeddingCacheStatusBypassed {
			t.Fatalf("SearchDetailed(%d) cache status = %q, want %q", i, response.Diagnostics.QueryEmbeddingCacheStatus, QueryEmbeddingCacheStatusBypassed)
		}
		if response.Diagnostics.QueryEmbeddingCacheHit {
			t.Fatalf("SearchDetailed(%d) unexpectedly reported cache hit", i)
		}
	}

	if got := embedder.CallCount() - callsAfterSeed; got != 2 {
		t.Fatalf("query embedder calls with cache bypassed = %d, want 2", got)
	}
}

func TestIntegrationNewRejectsQueryEmbeddingCacheWithoutCacheKeyer(t *testing.T) {
	ctx := context.Background()
	_, err := New(ctx, Config{
		DatabaseURL:             "postgres://demo",
		DefaultCollection:       "integration",
		EmbeddingDimensions:     256,
		Embedder:                &plainCountingEmbedder{dimensions: 256},
		QueryEmbeddingCacheSize: 8,
	})
	if err == nil {
		t.Fatal("expected New() to reject query cache without QueryEmbeddingCacheKeyer")
	}
	if !strings.Contains(err.Error(), "QueryEmbeddingCacheKeyer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegrationSearchDetailedUsesContextAwareCacheKeys(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	embedder := &scopedCountingEmbedder{
		base: &countingEmbedder{dimensions: 256},
	}
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:       "integration",
		EmbeddingDimensions:     256,
		Embedder:                embedder,
		QueryEmbeddingCacheSize: 8,
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-context-cache",
		Title:      "Context aware cache search",
		Content:    "Different request scopes should not share one cached query embedding.",
	}); err != nil {
		t.Fatalf("UpsertDocument() error = %v", err)
	}

	callsAfterSeed := embedder.CallCount()

	alphaCtx := context.WithValue(ctx, integrationQueryCacheScopeKey{}, "alpha")
	alphaFirst, err := store.SearchDetailed(alphaCtx, SearchRequest{
		Query: "scoped cache",
		Limit: 3,
		Mode:  SearchModeVector,
	})
	if err != nil {
		t.Fatalf("alpha first SearchDetailed() error = %v", err)
	}
	if alphaFirst.Diagnostics.QueryEmbeddingCacheHit {
		t.Fatal("expected first alpha search to miss cache")
	}
	if alphaFirst.Diagnostics.QueryEmbeddingCacheStatus != QueryEmbeddingCacheStatusMiss {
		t.Fatalf("alpha first cache status = %q, want %q", alphaFirst.Diagnostics.QueryEmbeddingCacheStatus, QueryEmbeddingCacheStatusMiss)
	}

	alphaSecond, err := store.SearchDetailed(alphaCtx, SearchRequest{
		Query: "scoped cache",
		Limit: 3,
		Mode:  SearchModeVector,
	})
	if err != nil {
		t.Fatalf("alpha second SearchDetailed() error = %v", err)
	}
	if !alphaSecond.Diagnostics.QueryEmbeddingCacheHit {
		t.Fatal("expected second alpha search to hit cache")
	}
	if alphaSecond.Diagnostics.QueryEmbeddingCacheStatus != QueryEmbeddingCacheStatusHit {
		t.Fatalf("alpha second cache status = %q, want %q", alphaSecond.Diagnostics.QueryEmbeddingCacheStatus, QueryEmbeddingCacheStatusHit)
	}

	betaCtx := context.WithValue(ctx, integrationQueryCacheScopeKey{}, "beta")
	betaFirst, err := store.SearchDetailed(betaCtx, SearchRequest{
		Query: "scoped cache",
		Limit: 3,
		Mode:  SearchModeVector,
	})
	if err != nil {
		t.Fatalf("beta first SearchDetailed() error = %v", err)
	}
	if betaFirst.Diagnostics.QueryEmbeddingCacheHit {
		t.Fatal("expected first beta search to miss cache because scope changed")
	}
	if betaFirst.Diagnostics.QueryEmbeddingCacheStatus != QueryEmbeddingCacheStatusMiss {
		t.Fatalf("beta first cache status = %q, want %q", betaFirst.Diagnostics.QueryEmbeddingCacheStatus, QueryEmbeddingCacheStatusMiss)
	}

	betaSecond, err := store.SearchDetailed(betaCtx, SearchRequest{
		Query: "scoped cache",
		Limit: 3,
		Mode:  SearchModeVector,
	})
	if err != nil {
		t.Fatalf("beta second SearchDetailed() error = %v", err)
	}
	if !betaSecond.Diagnostics.QueryEmbeddingCacheHit {
		t.Fatal("expected second beta search to hit cache")
	}
	if betaSecond.Diagnostics.QueryEmbeddingCacheStatus != QueryEmbeddingCacheStatusHit {
		t.Fatalf("beta second cache status = %q, want %q", betaSecond.Diagnostics.QueryEmbeddingCacheStatus, QueryEmbeddingCacheStatusHit)
	}

	if got := embedder.CallCount() - callsAfterSeed; got != 2 {
		t.Fatalf("query embedder calls across two scopes = %d, want 2", got)
	}
	if len(alphaSecond.Hits) == 0 || len(betaSecond.Hits) == 0 {
		t.Fatal("expected scoped searches to return hits")
	}
}

func TestIntegrationSearchDetailedConcurrentCacheReadsDoNotReembed(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	embedder := &countingEmbedder{dimensions: 256}
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:       "integration",
		EmbeddingDimensions:     256,
		Embedder:                embedder,
		QueryEmbeddingCacheSize: 8,
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-concurrent-cache",
		Title:      "Concurrent cache search",
		Content:    "Once the query embedding is warm, concurrent reads should reuse it safely.",
	}); err != nil {
		t.Fatalf("UpsertDocument() error = %v", err)
	}

	if _, err := store.SearchDetailed(ctx, SearchRequest{
		Query: "warm cache",
		Limit: 3,
		Mode:  SearchModeVector,
	}); err != nil {
		t.Fatalf("warm SearchDetailed() error = %v", err)
	}

	callsAfterWarm := embedder.CallCount()

	var wg sync.WaitGroup
	errCh := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response, err := store.SearchDetailed(ctx, SearchRequest{
				Query: "warm cache",
				Limit: 3,
				Mode:  SearchModeVector,
			})
			if err != nil {
				errCh <- err
				return
			}
			if !response.Diagnostics.QueryEmbeddingCacheHit {
				errCh <- errors.New("expected concurrent warm-cache search to hit cache")
				return
			}
			if response.Diagnostics.QueryEmbeddingCacheStatus != QueryEmbeddingCacheStatusHit {
				errCh <- errors.New("expected concurrent warm-cache search to report cache hit status")
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent SearchDetailed() error = %v", err)
		}
	}
	if got := embedder.CallCount() - callsAfterWarm; got != 0 {
		t.Fatalf("query embedder calls after warm-cache concurrent searches = %d, want 0", got)
	}
}

func TestIntegrationSearchDetailedConcurrentColdCacheMissesRemainSafe(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	embedder := &countingEmbedder{
		dimensions: 256,
		delay:      25 * time.Millisecond,
	}
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:       "integration",
		EmbeddingDimensions:     256,
		Embedder:                embedder,
		QueryEmbeddingCacheSize: 8,
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-cold-cache",
		Title:      "Cold cache search",
		Content:    "Concurrent cold misses may duplicate embedding work but must stay correct and race-free.",
	}); err != nil {
		t.Fatalf("UpsertDocument() error = %v", err)
	}

	callsAfterSeed := embedder.CallCount()

	const workers = 6
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response, err := store.SearchDetailed(ctx, SearchRequest{
				Query: "cold cache",
				Limit: 3,
				Mode:  SearchModeVector,
			})
			if err != nil {
				errCh <- err
				return
			}
			if len(response.Hits) == 0 {
				errCh <- errors.New("expected cold-cache concurrent search to return hits")
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent cold-cache SearchDetailed() error = %v", err)
		}
	}

	gotQueryCalls := embedder.CallCount() - callsAfterSeed
	if gotQueryCalls < 1 || gotQueryCalls > workers {
		t.Fatalf("query embedder calls after cold-cache concurrent searches = %d, want between 1 and %d", gotQueryCalls, workers)
	}

	response, err := store.SearchDetailed(ctx, SearchRequest{
		Query: "cold cache",
		Limit: 3,
		Mode:  SearchModeVector,
	})
	if err != nil {
		t.Fatalf("follow-up SearchDetailed() error = %v", err)
	}
	if !response.Diagnostics.QueryEmbeddingCacheHit {
		t.Fatal("expected follow-up cold-cache search to hit cache after warm-up")
	}
	if response.Diagnostics.QueryEmbeddingCacheStatus != QueryEmbeddingCacheStatusHit {
		t.Fatalf("follow-up cache status = %q, want %q", response.Diagnostics.QueryEmbeddingCacheStatus, QueryEmbeddingCacheStatusHit)
	}
}

func TestIntegrationSearchDetailedHonorsContextDeadlineDuringQueryEmbedding(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	embedder := &countingEmbedder{
		dimensions: 256,
		delay:      50 * time.Millisecond,
	}
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            embedder,
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-deadline",
		Title:      "Deadline search",
		Content:    "Slow query embedding should still respect the caller deadline.",
	}); err != nil {
		t.Fatalf("UpsertDocument() error = %v", err)
	}

	deadlineCtx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()

	_, err := store.SearchDetailed(deadlineCtx, SearchRequest{
		Query: "deadline pressure",
		Limit: 3,
		Mode:  SearchModeVector,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
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

func TestIntegrationMigrateUpgradesOlderSchemaWithoutLosingData(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	seedLegacyIntegrationSchema(t, databaseURL, schema, 256, 2)

	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	gotVersions := appliedMigrationVersions(t, ctx, store.pool)
	wantVersions := []int64{1, 2, 3, 4}
	if !slices.Equal(gotVersions, wantVersions) {
		t.Fatalf("migration versions = %v, want %v", gotVersions, wantVersions)
	}

	hits, err := store.Search(ctx, SearchRequest{
		Query: "upgrade regression",
		Limit: 3,
		Mode:  SearchModeKeyword,
		MetadataFilter: map[string]any{
			"tenant": "legacy",
		},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected migrated legacy document to remain searchable")
	}
	if hits[0].DocumentID != "legacy-doc" {
		t.Fatalf("unexpected migrated top hit: %+v", hits[0])
	}

	if _, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "fresh-doc",
		Title:      "Fresh document",
		Content:    "New writes should still work after upgrading the schema.",
	}); err != nil {
		t.Fatalf("UpsertDocument() after upgrade error = %v", err)
	}

	var documentCount int
	if err := store.pool.QueryRow(ctx, `
SELECT count(*)
FROM kb_documents
WHERE collection = $1
`, "integration").Scan(&documentCount); err != nil {
		t.Fatalf("count documents after upgrade: %v", err)
	}
	if documentCount != 2 {
		t.Fatalf("document count = %d, want 2", documentCount)
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

func TestIntegrationUpsertNoopSkipsSplitterEmbedderAndChunkRewrite(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	splitter := &countingSplitter{
		chunks: []ChunkDraft{
			{Ordinal: 0, Content: "first chunk"},
			{Ordinal: 1, Content: "second chunk"},
		},
	}
	embedder := &countingEmbedder{dimensions: 256}
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            embedder,
		Splitter:            splitter,
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	req := UpsertDocumentRequest{
		DocumentID: "doc-noop",
		Title:      "No-op document",
		Content:    "stable content",
		SourceURI:  "https://example.com/noop",
		Tags:       []string{"alpha", "stable"},
		Metadata: map[string]any{
			"tenant": "alpha",
			"tier":   "gold",
		},
	}

	stats, err := store.UpsertDocument(ctx, req)
	if err != nil {
		t.Fatalf("first UpsertDocument() error = %v", err)
	}
	if stats.ChunkCount != 2 {
		t.Fatalf("first chunk count = %d, want 2", stats.ChunkCount)
	}

	before := loadIntegrationDocumentState(t, ctx, store.pool, "integration", "doc-noop")
	if splitter.CallCount() != 1 {
		t.Fatalf("splitter calls after first upsert = %d, want 1", splitter.CallCount())
	}
	if embedder.CallCount() != 1 {
		t.Fatalf("embedder calls after first upsert = %d, want 1", embedder.CallCount())
	}

	time.Sleep(20 * time.Millisecond)

	stats, err = store.UpsertDocument(ctx, req)
	if err != nil {
		t.Fatalf("second UpsertDocument() error = %v", err)
	}
	if stats.ChunkCount != 2 {
		t.Fatalf("second chunk count = %d, want 2", stats.ChunkCount)
	}
	if splitter.CallCount() != 1 {
		t.Fatalf("splitter calls after no-op upsert = %d, want 1", splitter.CallCount())
	}
	if embedder.CallCount() != 1 {
		t.Fatalf("embedder calls after no-op upsert = %d, want 1", embedder.CallCount())
	}

	after := loadIntegrationDocumentState(t, ctx, store.pool, "integration", "doc-noop")
	if !after.DocumentUpdatedAt.Equal(before.DocumentUpdatedAt) {
		t.Fatalf("document updated_at changed on no-op: before=%v after=%v", before.DocumentUpdatedAt, after.DocumentUpdatedAt)
	}
	assertChunkRowsPreserved(t, before.Chunks, after.Chunks)
}

func TestIntegrationUpsertMetadataRefreshUpdatesDocumentAndChunksWithoutReembedding(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	splitter := &countingSplitter{
		chunks: []ChunkDraft{
			{Ordinal: 0, Content: "first chunk"},
			{Ordinal: 1, Content: "second chunk"},
		},
	}
	embedder := &countingEmbedder{dimensions: 256}
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            embedder,
		Splitter:            splitter,
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	req := UpsertDocumentRequest{
		DocumentID: "doc-refresh",
		Title:      "Original title",
		Content:    "stable content",
		SourceURI:  "https://example.com/original",
		Tags:       []string{"alpha"},
		Metadata: map[string]any{
			"tenant": "alpha",
			"tier":   "gold",
		},
	}

	if _, err := store.UpsertDocument(ctx, req); err != nil {
		t.Fatalf("first UpsertDocument() error = %v", err)
	}
	before := loadIntegrationDocumentState(t, ctx, store.pool, "integration", "doc-refresh")

	time.Sleep(20 * time.Millisecond)

	req.Title = "Updated title"
	req.SourceURI = "https://example.com/updated"
	req.Tags = []string{"beta", "alpha"}
	req.Metadata = map[string]any{
		"tenant": "alpha",
		"tier":   "platinum",
		"region": "cn",
	}

	stats, err := store.UpsertDocument(ctx, req)
	if err != nil {
		t.Fatalf("metadata refresh UpsertDocument() error = %v", err)
	}
	if stats.ChunkCount != 2 {
		t.Fatalf("metadata refresh chunk count = %d, want 2", stats.ChunkCount)
	}
	if splitter.CallCount() != 1 {
		t.Fatalf("splitter calls after metadata refresh = %d, want 1", splitter.CallCount())
	}
	if embedder.CallCount() != 1 {
		t.Fatalf("embedder calls after metadata refresh = %d, want 1", embedder.CallCount())
	}

	after := loadIntegrationDocumentState(t, ctx, store.pool, "integration", "doc-refresh")
	if !after.DocumentUpdatedAt.After(before.DocumentUpdatedAt) {
		t.Fatalf("document updated_at did not advance on metadata refresh: before=%v after=%v", before.DocumentUpdatedAt, after.DocumentUpdatedAt)
	}
	if after.ContentHash != before.ContentHash {
		t.Fatalf("content hash changed on metadata refresh: before=%s after=%s", before.ContentHash, after.ContentHash)
	}
	if after.Title != req.Title {
		t.Fatalf("document title = %q, want %q", after.Title, req.Title)
	}
	if after.SourceURI != req.SourceURI {
		t.Fatalf("document source_uri = %q, want %q", after.SourceURI, req.SourceURI)
	}
	if !slices.Equal(after.Tags, []string{"alpha", "beta"}) {
		t.Fatalf("document tags = %v, want [alpha beta]", after.Tags)
	}
	assertMetadataEqual(t, after.Metadata, req.Metadata)

	if len(after.Chunks) != len(before.Chunks) {
		t.Fatalf("chunk count after metadata refresh = %d, want %d", len(after.Chunks), len(before.Chunks))
	}
	for i := range after.Chunks {
		if after.Chunks[i].ID != before.Chunks[i].ID {
			t.Fatalf("chunk %d was rewritten during metadata refresh: before=%d after=%d", i, before.Chunks[i].ID, after.Chunks[i].ID)
		}
		if !after.Chunks[i].UpdatedAt.After(before.Chunks[i].UpdatedAt) {
			t.Fatalf("chunk %d updated_at did not advance on metadata refresh: before=%v after=%v", i, before.Chunks[i].UpdatedAt, after.Chunks[i].UpdatedAt)
		}
		if after.Chunks[i].Title != req.Title {
			t.Fatalf("chunk %d title = %q, want %q", i, after.Chunks[i].Title, req.Title)
		}
		if after.Chunks[i].SourceURI != req.SourceURI {
			t.Fatalf("chunk %d source_uri = %q, want %q", i, after.Chunks[i].SourceURI, req.SourceURI)
		}
		if !slices.Equal(after.Chunks[i].Tags, []string{"alpha", "beta"}) {
			t.Fatalf("chunk %d tags = %v, want [alpha beta]", i, after.Chunks[i].Tags)
		}
		assertMetadataEqual(t, after.Chunks[i].Metadata, req.Metadata)
	}
}

func TestIntegrationReindexDocumentForcesRecipeRolloutWhenContentIsUnchanged(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)

	embedderA := &countingEmbedder{dimensions: 256}
	splitterA := &countingSplitter{
		chunks: []ChunkDraft{
			{Ordinal: 0, Content: "recipe-a chunk"},
		},
	}
	storeA := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            embedderA,
		Splitter:            splitterA,
	})
	if err := storeA.Migrate(ctx); err != nil {
		t.Fatalf("storeA Migrate() error = %v", err)
	}

	req := UpsertDocumentRequest{
		DocumentID: "doc-reindex",
		Title:      "Recipe rollout",
		Content:    "same content",
		SourceURI:  "https://example.com/reindex",
		Tags:       []string{"alpha"},
		Metadata: map[string]any{
			"tenant": "alpha",
		},
	}
	if _, err := storeA.UpsertDocument(ctx, req); err != nil {
		t.Fatalf("storeA UpsertDocument() error = %v", err)
	}

	before := loadIntegrationDocumentState(t, ctx, storeA.pool, "integration", "doc-reindex")
	if len(before.Chunks) != 1 {
		t.Fatalf("initial chunk count = %d, want 1", len(before.Chunks))
	}

	embedderB := &countingEmbedder{dimensions: 256}
	splitterB := &countingSplitter{
		chunks: []ChunkDraft{
			{Ordinal: 0, Content: "recipe-b chunk one"},
			{Ordinal: 1, Content: "recipe-b chunk two"},
		},
	}
	storeB := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            embedderB,
		Splitter:            splitterB,
	})
	if err := storeB.Migrate(ctx); err != nil {
		t.Fatalf("storeB Migrate() error = %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	stats, err := storeB.UpsertDocument(ctx, req)
	if err != nil {
		t.Fatalf("storeB UpsertDocument() error = %v", err)
	}
	if stats.ChunkCount != 1 {
		t.Fatalf("UpsertDocument() during recipe rollout chunk count = %d, want 1", stats.ChunkCount)
	}
	if splitterB.CallCount() != 0 {
		t.Fatalf("splitter calls during recipe-rollout noop = %d, want 0", splitterB.CallCount())
	}
	if embedderB.CallCount() != 0 {
		t.Fatalf("embedder calls during recipe-rollout noop = %d, want 0", embedderB.CallCount())
	}

	afterNoop := loadIntegrationDocumentState(t, ctx, storeB.pool, "integration", "doc-reindex")
	if !afterNoop.DocumentUpdatedAt.Equal(before.DocumentUpdatedAt) {
		t.Fatalf("document updated_at changed on recipe-rollout noop: before=%v after=%v", before.DocumentUpdatedAt, afterNoop.DocumentUpdatedAt)
	}
	assertChunkRowsPreserved(t, before.Chunks, afterNoop.Chunks)

	time.Sleep(20 * time.Millisecond)

	stats, err = storeB.ReindexDocument(ctx, req)
	if err != nil {
		t.Fatalf("ReindexDocument() error = %v", err)
	}
	if stats.ChunkCount != 2 {
		t.Fatalf("ReindexDocument() chunk count = %d, want 2", stats.ChunkCount)
	}
	if splitterB.CallCount() != 1 {
		t.Fatalf("splitter calls after ReindexDocument() = %d, want 1", splitterB.CallCount())
	}
	if embedderB.CallCount() != 1 {
		t.Fatalf("embedder calls after ReindexDocument() = %d, want 1", embedderB.CallCount())
	}

	afterReindex := loadIntegrationDocumentState(t, ctx, storeB.pool, "integration", "doc-reindex")
	if len(afterReindex.Chunks) != 2 {
		t.Fatalf("chunk count after ReindexDocument() = %d, want 2", len(afterReindex.Chunks))
	}
	if !afterReindex.DocumentUpdatedAt.After(afterNoop.DocumentUpdatedAt) {
		t.Fatalf("document updated_at did not advance on ReindexDocument(): before=%v after=%v", afterNoop.DocumentUpdatedAt, afterReindex.DocumentUpdatedAt)
	}
	if len(afterNoop.Chunks) >= len(afterReindex.Chunks) && afterNoop.Chunks[0].ID == afterReindex.Chunks[0].ID {
		t.Fatalf("expected chunk rows to be rebuilt on ReindexDocument(), first chunk id stayed %d", afterReindex.Chunks[0].ID)
	}
	if afterReindex.Chunks[0].Content != "recipe-b chunk one" || afterReindex.Chunks[1].Content != "recipe-b chunk two" {
		t.Fatalf("unexpected chunk contents after ReindexDocument(): %+v", afterReindex.Chunks)
	}
}

func TestIntegrationCommitReindexDocumentWriteRetriesWhenExpectedSnapshotIsStale(t *testing.T) {
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

	req := store.normalizeDocumentRequest(UpsertDocumentRequest{
		DocumentID: "doc-stale-reindex",
		Title:      "Stale reindex",
		Content:    "same content",
		Metadata: map[string]any{
			"tenant": "alpha",
		},
	})
	if _, err := store.UpsertDocument(ctx, req); err != nil {
		t.Fatalf("seed UpsertDocument() error = %v", err)
	}

	expected, err := loadDocumentSnapshot(ctx, store.pool, req.Collection, req.DocumentID, false)
	if err != nil {
		t.Fatalf("loadDocumentSnapshot() error = %v", err)
	}

	contentHash := hashText(req.Content)
	metadataJSON, err := canonicalMetadataJSON(req.Metadata)
	if err != nil {
		t.Fatalf("canonicalMetadataJSON() error = %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	freshPrepared := preparedDocumentWrite{
		chunks: []ChunkDraft{
			{Ordinal: 0, Content: "fresh rebuilt chunk"},
		},
		vectors: [][]float32{
			makeVector(256, 1),
		},
	}
	_, shouldRestart, err := store.commitReindexDocumentWrite(ctx, req, contentHash, metadataJSON, freshPrepared, expected, true)
	if err != nil {
		t.Fatalf("fresh commitReindexDocumentWrite() error = %v", err)
	}
	if shouldRestart {
		t.Fatal("fresh commitReindexDocumentWrite() unexpectedly requested restart")
	}

	afterFresh := loadIntegrationDocumentState(t, ctx, store.pool, req.Collection, req.DocumentID)
	if len(afterFresh.Chunks) != 1 {
		t.Fatalf("chunk count after fresh reindex = %d, want 1", len(afterFresh.Chunks))
	}
	if afterFresh.Chunks[0].Content != "fresh rebuilt chunk" {
		t.Fatalf("fresh chunk content = %q, want %q", afterFresh.Chunks[0].Content, "fresh rebuilt chunk")
	}

	stalePrepared := preparedDocumentWrite{
		chunks: []ChunkDraft{
			{Ordinal: 0, Content: "stale rebuilt chunk"},
		},
		vectors: [][]float32{
			makeVector(256, 2),
		},
	}
	_, shouldRestart, err = store.commitReindexDocumentWrite(ctx, req, contentHash, metadataJSON, stalePrepared, expected, true)
	if err != nil {
		t.Fatalf("stale commitReindexDocumentWrite() error = %v", err)
	}
	if !shouldRestart {
		t.Fatal("stale commitReindexDocumentWrite() should have requested restart")
	}

	afterStaleAttempt := loadIntegrationDocumentState(t, ctx, store.pool, req.Collection, req.DocumentID)
	if !afterStaleAttempt.DocumentUpdatedAt.Equal(afterFresh.DocumentUpdatedAt) {
		t.Fatalf("document updated_at changed on stale reindex attempt: before=%v after=%v", afterFresh.DocumentUpdatedAt, afterStaleAttempt.DocumentUpdatedAt)
	}
	if afterStaleAttempt.Chunks[0].Content != "fresh rebuilt chunk" {
		t.Fatalf("stale reindex overwrote fresh chunk content: got %q", afterStaleAttempt.Chunks[0].Content)
	}
}

func TestIntegrationUpsertRejectsEmbedderVectorCountMismatch(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder: staticEmbedder{
			vectors: [][]float32{},
		},
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	_, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-1",
		Title:      "Mismatch",
		Content:    "embedder count mismatch should fail loudly",
	})
	if err == nil {
		t.Fatal("expected vector count mismatch")
	}
	if !strings.Contains(err.Error(), "vectors for") {
		t.Fatalf("expected vector count mismatch error, got %v", err)
	}
}

func TestIntegrationUpsertRejectsEmbedderVectorDimensionMismatch(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder: staticEmbedder{
			vectors: [][]float32{
				{1, 0},
			},
		},
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	_, err := store.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-1",
		Title:      "Dimension mismatch",
		Content:    "embedder dimension mismatch should fail loudly",
	})
	if err == nil {
		t.Fatal("expected vector dimension mismatch")
	}
	if !strings.Contains(err.Error(), "dimension mismatch") {
		t.Fatalf("expected dimension mismatch error, got %v", err)
	}
}

func TestIntegrationSearchRejectsQueryEmbedderCountMismatch(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder: staticEmbedder{
			vectors: [][]float32{
				makeVector(256, 0),
				makeVector(256, 1),
			},
		},
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	_, err := store.Search(ctx, SearchRequest{
		Query: "vector only query",
		Limit: 1,
		Mode:  SearchModeVector,
	})
	if err == nil {
		t.Fatal("expected query vector count mismatch")
	}
	if !strings.Contains(err.Error(), "query vectors") {
		t.Fatalf("expected query vector count error, got %v", err)
	}
}

func TestIntegrationSearchRejectsQueryEmbedderDimensionMismatch(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)

	seedStore := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
	})
	if err := seedStore.Migrate(ctx); err != nil {
		t.Fatalf("seedStore Migrate() error = %v", err)
	}
	if _, err := seedStore.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-query-dim",
		Title:      "Query dimension mismatch",
		Content:    "query embedding dimension mismatch should fail before SQL",
	}); err != nil {
		t.Fatalf("seedStore UpsertDocument() error = %v", err)
	}

	embedder := &countingEmbedder{dimensions: 128}
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            embedder,
	})
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("store Migrate() error = %v", err)
	}

	_, err := store.Search(ctx, SearchRequest{
		Query: "vector only query",
		Limit: 1,
		Mode:  SearchModeVector,
	})
	if err == nil {
		t.Fatal("expected query vector dimension mismatch")
	}
	if !strings.Contains(err.Error(), "query vector dimension mismatch") {
		t.Fatalf("expected query vector dimension mismatch error, got %v", err)
	}
	if got := embedder.CallCount(); got != 1 {
		t.Fatalf("query embedder calls = %d, want 1", got)
	}
}

func TestIntegrationSearchDoesNotCacheWrongDimensionQueryEmbedding(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)

	seedStore := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
	})
	if err := seedStore.Migrate(ctx); err != nil {
		t.Fatalf("seedStore Migrate() error = %v", err)
	}
	if _, err := seedStore.UpsertDocument(ctx, UpsertDocumentRequest{
		DocumentID: "doc-query-cache-dim",
		Title:      "Cached query dimension mismatch",
		Content:    "wrong-dimension query embeddings must not poison the cache",
	}); err != nil {
		t.Fatalf("seedStore UpsertDocument() error = %v", err)
	}

	embedder := &countingEmbedder{dimensions: 128}
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:       "integration",
		EmbeddingDimensions:     256,
		Embedder:                embedder,
		QueryEmbeddingCacheSize: 8,
	})
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("store Migrate() error = %v", err)
	}

	for i := 0; i < 2; i++ {
		_, err := store.SearchDetailed(ctx, SearchRequest{
			Query: "cached vector only query",
			Limit: 1,
			Mode:  SearchModeVector,
		})
		if err == nil {
			t.Fatalf("SearchDetailed(%d) expected query vector dimension mismatch", i)
		}
		if !strings.Contains(err.Error(), "query vector dimension mismatch") {
			t.Fatalf("SearchDetailed(%d) expected query vector dimension mismatch error, got %v", i, err)
		}
	}
	if got := embedder.CallCount(); got != 2 {
		t.Fatalf("query embedder calls after repeated wrong-dimension searches = %d, want 2", got)
	}
}

func TestIntegrationSearchHonorsCanceledContext(t *testing.T) {
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

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err := store.Search(canceledCtx, SearchRequest{
		Query: "anything",
		Limit: 1,
		Mode:  SearchModeVector,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
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

type countingSplitter struct {
	mu     sync.Mutex
	calls  int
	chunks []ChunkDraft
	err    error
}

func (s *countingSplitter) Split(text string) ([]ChunkDraft, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}

	out := make([]ChunkDraft, len(s.chunks))
	copy(out, s.chunks)
	return out, nil
}

func (s *countingSplitter) CallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func integrationDatabaseURL(t *testing.T) string {
	return testdb.DatabaseURL(t)
}

func createIntegrationSchema(t *testing.T, databaseURL string) string {
	return testdb.CreateSchema(t, databaseURL, "simplykb_test")
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

func seedLegacyIntegrationSchema(t *testing.T, databaseURL string, schema string, dimensions int, lastVersion int64) {
	t.Helper()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURLWithSearchPath(t, databaseURL, schema))
	if err != nil {
		t.Fatalf("connect legacy pool: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, bootstrapMigrationSQL()); err != nil {
		t.Fatalf("bootstrap legacy migrations table: %v", err)
	}

	for _, migration := range schemaMigrations(dimensions) {
		if migration.version > lastVersion {
			break
		}
		if _, err := pool.Exec(ctx, migration.sql); err != nil {
			t.Fatalf("apply legacy migration %d (%s): %v", migration.version, migration.name, err)
		}
		if _, err := pool.Exec(ctx, `
INSERT INTO kb_schema_migrations (version, name)
VALUES ($1, $2)
`, migration.version, migration.name); err != nil {
			t.Fatalf("record legacy migration %d (%s): %v", migration.version, migration.name, err)
		}
	}

	const legacyContent = "Legacy upgrade regression content stays searchable after migrations."
	const legacyMetadata = `{"tenant":"legacy"}`

	var internalDocumentID int64
	err = pool.QueryRow(ctx, `
INSERT INTO kb_documents (
    collection,
    external_id,
    title,
    source_uri,
    tags,
    metadata,
    content_hash,
    chunk_count
) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
RETURNING id
`, "integration", "legacy-doc", "Legacy document", "", []string{"legacy"}, legacyMetadata, hashText(legacyContent), 1).Scan(&internalDocumentID)
	if err != nil {
		t.Fatalf("insert legacy document: %v", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO kb_chunks (
    collection,
    document_id,
    document_external_id,
    chunk_key,
    chunk_no,
    title,
    content,
    source_uri,
    tags,
    metadata,
    embedding
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::vector)
`, "integration", internalDocumentID, "legacy-doc", chunkKey("legacy-doc", 0), 0, "Legacy document", legacyContent, "", []string{"legacy"}, legacyMetadata, vectorLiteral(makeVector(dimensions, 0))); err != nil {
		t.Fatalf("insert legacy chunk: %v", err)
	}
}

func appliedMigrationVersions(t *testing.T, ctx context.Context, pool *pgxpool.Pool) []int64 {
	t.Helper()

	rows, err := pool.Query(ctx, `
SELECT version
FROM kb_schema_migrations
ORDER BY version
`)
	if err != nil {
		t.Fatalf("query migration versions: %v", err)
	}
	defer rows.Close()

	var versions []int64
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			t.Fatalf("scan migration version: %v", err)
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate migration versions: %v", err)
	}
	return versions
}

func databaseURLWithSearchPath(t *testing.T, databaseURL string, schema string) string {
	return testdb.URLWithSearchPath(t, databaseURL, schema)
}

type staticEmbedder struct {
	vectors [][]float32
	err     error
}

func (e staticEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e.err != nil {
		return nil, e.err
	}

	out := make([][]float32, len(e.vectors))
	for i, vector := range e.vectors {
		out[i] = append([]float32(nil), vector...)
	}
	return out, nil
}

func makeVector(dimensions int, hotIndex int) []float32 {
	vector := make([]float32, dimensions)
	vector[hotIndex] = 1
	return vector
}

type countingEmbedder struct {
	mu         sync.Mutex
	calls      int
	dimensions int
	err        error
	delay      time.Duration
}

func (e *countingEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	e.mu.Lock()
	if e.err != nil {
		e.mu.Unlock()
		return nil, e.err
	}

	e.calls++
	callNumber := e.calls
	delay := e.delay
	dimensions := e.dimensions
	e.mu.Unlock()

	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = makeVector(dimensions, (callNumber+i)%dimensions)
	}
	return out, nil
}

func (e *countingEmbedder) QueryEmbeddingCacheKey(ctx context.Context, normalizedQuery string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	if bypass, _ := ctx.Value(integrationQueryCacheBypassKey{}).(bool); bypass {
		return "", false, nil
	}
	return normalizedQuery, true, nil
}

func (e *countingEmbedder) CallCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

type integrationQueryCacheScopeKey struct{}

type integrationQueryCacheBypassKey struct{}

type scopedCountingEmbedder struct {
	base *countingEmbedder
}

func (e *scopedCountingEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return e.base.Embed(ctx, texts)
}

func (e *scopedCountingEmbedder) QueryEmbeddingCacheKey(ctx context.Context, normalizedQuery string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	if bypass, _ := ctx.Value(integrationQueryCacheBypassKey{}).(bool); bypass {
		return "", false, nil
	}
	scope, _ := ctx.Value(integrationQueryCacheScopeKey{}).(string)
	if scope == "" {
		scope = "default"
	}
	return scope + ":" + normalizedQuery, true, nil
}

func (e *scopedCountingEmbedder) CallCount() int {
	return e.base.CallCount()
}

type integrationDocumentState struct {
	InternalID        int64
	Title             string
	SourceURI         string
	Tags              []string
	Metadata          map[string]any
	ContentHash       string
	ChunkCount        int
	DocumentUpdatedAt time.Time
	Chunks            []integrationChunkState
}

type integrationChunkState struct {
	ID        int64
	ChunkNo   int
	ChunkKey  string
	Title     string
	Content   string
	SourceURI string
	Tags      []string
	Metadata  map[string]any
	UpdatedAt time.Time
}

func loadIntegrationDocumentState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, collection string, documentID string) integrationDocumentState {
	t.Helper()

	var (
		state         integrationDocumentState
		metadataBytes []byte
	)
	err := pool.QueryRow(ctx, `
SELECT id, title, source_uri, tags, metadata, content_hash, chunk_count, updated_at
FROM kb_documents
WHERE collection = $1 AND external_id = $2
`, collection, documentID).Scan(
		&state.InternalID,
		&state.Title,
		&state.SourceURI,
		&state.Tags,
		&metadataBytes,
		&state.ContentHash,
		&state.ChunkCount,
		&state.DocumentUpdatedAt,
	)
	if err != nil {
		t.Fatalf("load document state: %v", err)
	}
	state.Metadata = decodeMetadata(t, metadataBytes)

	rows, err := pool.Query(ctx, `
SELECT id, chunk_no, chunk_key, title, content, source_uri, tags, metadata, updated_at
FROM kb_chunks
WHERE collection = $1 AND document_external_id = $2
ORDER BY chunk_no
`, collection, documentID)
	if err != nil {
		t.Fatalf("query chunk state: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			chunk         integrationChunkState
			metadataBytes []byte
		)
		if err := rows.Scan(
			&chunk.ID,
			&chunk.ChunkNo,
			&chunk.ChunkKey,
			&chunk.Title,
			&chunk.Content,
			&chunk.SourceURI,
			&chunk.Tags,
			&metadataBytes,
			&chunk.UpdatedAt,
		); err != nil {
			t.Fatalf("scan chunk state: %v", err)
		}
		chunk.Metadata = decodeMetadata(t, metadataBytes)
		state.Chunks = append(state.Chunks, chunk)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate chunk state: %v", err)
	}
	return state
}

func decodeMetadata(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	if len(raw) == 0 {
		return map[string]any{}
	}

	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func assertChunkRowsPreserved(t *testing.T, before []integrationChunkState, after []integrationChunkState) {
	t.Helper()

	if len(after) != len(before) {
		t.Fatalf("chunk count changed: before=%d after=%d", len(before), len(after))
	}
	for i := range before {
		if after[i].ID != before[i].ID {
			t.Fatalf("chunk %d id changed: before=%d after=%d", i, before[i].ID, after[i].ID)
		}
		if after[i].ChunkKey != before[i].ChunkKey {
			t.Fatalf("chunk %d key changed: before=%q after=%q", i, before[i].ChunkKey, after[i].ChunkKey)
		}
		if !after[i].UpdatedAt.Equal(before[i].UpdatedAt) {
			t.Fatalf("chunk %d updated_at changed: before=%v after=%v", i, before[i].UpdatedAt, after[i].UpdatedAt)
		}
	}
}

func assertMetadataEqual(t *testing.T, got map[string]any, want map[string]any) {
	t.Helper()

	gotJSON := canonicalMetadataJSONString(t, got)
	wantJSON := canonicalMetadataJSONString(t, maps.Clone(want))
	if gotJSON != wantJSON {
		t.Fatalf("metadata mismatch: got=%s want=%s", gotJSON, wantJSON)
	}
}

func assertSearchHitsEqual(t *testing.T, got []SearchHit, want []SearchHit) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("search hits length = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i].Collection != want[i].Collection ||
			got[i].DocumentID != want[i].DocumentID ||
			got[i].ChunkID != want[i].ChunkID ||
			got[i].ChunkNumber != want[i].ChunkNumber ||
			got[i].Title != want[i].Title ||
			got[i].Content != want[i].Content ||
			got[i].Snippet != want[i].Snippet ||
			got[i].SourceURI != want[i].SourceURI ||
			!slices.Equal(got[i].Tags, want[i].Tags) ||
			got[i].Score != want[i].Score ||
			got[i].KeywordScore != want[i].KeywordScore ||
			got[i].VectorScore != want[i].VectorScore {
			t.Fatalf("search hit %d mismatch:\n got=%+v\nwant=%+v", i, got[i], want[i])
		}
		assertMetadataEqual(t, got[i].Metadata, want[i].Metadata)
	}
}

func canonicalMetadataJSONString(t *testing.T, metadata map[string]any) string {
	t.Helper()
	bytes, err := json.Marshal(normalizeMetadata(metadata))
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	return string(bytes)
}
