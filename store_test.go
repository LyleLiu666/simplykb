package simplykb

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
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

func TestConfigValidateRejectsNegativeQueryEmbeddingCacheSize(t *testing.T) {
	cfg := Config{
		DatabaseURL:             "postgres://demo",
		EmbeddingDimensions:     64,
		Embedder:                NewHashEmbedder(64),
		QueryEmbeddingCacheSize: -1,
	}
	err := cfg.normalized().validate()
	if err == nil {
		t.Fatal("expected negative query embedding cache size to fail validation")
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

func TestConfigValidateRejectsQueryEmbeddingCacheWithoutCacheKeyer(t *testing.T) {
	cfg := Config{
		DatabaseURL:             "postgres://demo",
		EmbeddingDimensions:     64,
		Embedder:                &plainCountingEmbedder{dimensions: 64},
		QueryEmbeddingCacheSize: 8,
	}

	err := cfg.normalized().validate()
	if err == nil {
		t.Fatal("expected query embedding cache without QueryEmbeddingCacheKeyer to fail validation")
	}
	if !strings.Contains(err.Error(), "QueryEmbeddingCacheKeyer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchQueryVectorRejectsCacheWithoutCacheKeyer(t *testing.T) {
	embedder := &plainCountingEmbedder{dimensions: 4}
	store := &Store{
		cfg: Config{
			EmbeddingDimensions: 4,
			Embedder:            embedder,
		}.normalized(),
		queryCache: newQueryEmbeddingCache(8),
	}

	_, cacheStatus, err := store.searchQueryVector(context.Background(), "same query")
	if err == nil {
		t.Fatal("expected searchQueryVector() to reject cache without QueryEmbeddingCacheKeyer")
	}
	if cacheStatus != QueryEmbeddingCacheStatusNotApplicable {
		t.Fatalf("cache status = %q, want %q on contract error", cacheStatus, QueryEmbeddingCacheStatusNotApplicable)
	}
	if !strings.Contains(err.Error(), "QueryEmbeddingCacheKeyer") {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := embedder.CallCount(); got != 0 {
		t.Fatalf("embedder calls = %d, want 0 when cache contract is invalid", got)
	}
}

func TestSearchQueryVectorUsesEmbedderSuppliedCacheKey(t *testing.T) {
	embedder := &cacheAwareCountingEmbedder{dimensions: 4}
	store := &Store{
		cfg: Config{
			EmbeddingDimensions: 4,
			Embedder:            embedder,
		}.normalized(),
		queryCache: newQueryEmbeddingCache(8),
	}

	alphaCtx := context.WithValue(context.Background(), queryCacheScopeKey{}, "alpha")
	alphaFirst, cacheStatus, err := store.searchQueryVector(alphaCtx, "same query")
	if err != nil {
		t.Fatalf("alpha first searchQueryVector() error = %v", err)
	}
	if cacheStatus != QueryEmbeddingCacheStatusMiss {
		t.Fatalf("alpha first cache status = %q, want %q", cacheStatus, QueryEmbeddingCacheStatusMiss)
	}

	alphaSecond, cacheStatus, err := store.searchQueryVector(alphaCtx, "same query")
	if err != nil {
		t.Fatalf("alpha second searchQueryVector() error = %v", err)
	}
	if cacheStatus != QueryEmbeddingCacheStatusHit {
		t.Fatalf("alpha second cache status = %q, want %q", cacheStatus, QueryEmbeddingCacheStatusHit)
	}

	betaCtx := context.WithValue(context.Background(), queryCacheScopeKey{}, "beta")
	betaFirst, cacheStatus, err := store.searchQueryVector(betaCtx, "same query")
	if err != nil {
		t.Fatalf("beta first searchQueryVector() error = %v", err)
	}
	if cacheStatus != QueryEmbeddingCacheStatusMiss {
		t.Fatalf("beta first cache status = %q, want %q", cacheStatus, QueryEmbeddingCacheStatusMiss)
	}

	betaSecond, cacheStatus, err := store.searchQueryVector(betaCtx, "same query")
	if err != nil {
		t.Fatalf("beta second searchQueryVector() error = %v", err)
	}
	if cacheStatus != QueryEmbeddingCacheStatusHit {
		t.Fatalf("beta second cache status = %q, want %q", cacheStatus, QueryEmbeddingCacheStatusHit)
	}

	if got := embedder.CallCount(); got != 2 {
		t.Fatalf("embedder calls = %d, want 2", got)
	}
	if !slices.Equal(alphaFirst, alphaSecond) {
		t.Fatal("expected alpha cache hit to reuse alpha vector")
	}
	if !slices.Equal(betaFirst, betaSecond) {
		t.Fatal("expected beta cache hit to reuse beta vector")
	}
	if slices.Equal(alphaFirst, betaFirst) {
		t.Fatal("expected different cache scopes to keep vectors separate")
	}
}

func TestSearchQueryVectorAllowsEmbedderToBypassCache(t *testing.T) {
	embedder := &cacheAwareCountingEmbedder{dimensions: 4}
	store := &Store{
		cfg: Config{
			EmbeddingDimensions: 4,
			Embedder:            embedder,
		}.normalized(),
		queryCache: newQueryEmbeddingCache(8),
	}

	bypassCtx := context.WithValue(context.Background(), queryCacheBypassKey{}, true)
	for i := 0; i < 2; i++ {
		_, cacheStatus, err := store.searchQueryVector(bypassCtx, "same query")
		if err != nil {
			t.Fatalf("searchQueryVector(%d) error = %v", i, err)
		}
		if cacheStatus != QueryEmbeddingCacheStatusBypassed {
			t.Fatalf("searchQueryVector(%d) cache status = %q, want %q", i, cacheStatus, QueryEmbeddingCacheStatusBypassed)
		}
	}

	if got := embedder.CallCount(); got != 2 {
		t.Fatalf("embedder calls = %d, want 2", got)
	}
}

func TestSearchQueryVectorPropagatesCacheKeyResolutionErrors(t *testing.T) {
	embedder := &cacheAwareCountingEmbedder{
		dimensions:  4,
		cacheKeyErr: errors.New("boom"),
	}
	store := &Store{
		cfg: Config{
			EmbeddingDimensions: 4,
			Embedder:            embedder,
		}.normalized(),
		queryCache: newQueryEmbeddingCache(8),
	}

	_, cacheStatus, err := store.searchQueryVector(context.Background(), "same query")
	if err == nil {
		t.Fatal("expected cache key resolution error")
	}
	if cacheStatus != QueryEmbeddingCacheStatusNotApplicable {
		t.Fatalf("cache status = %q, want %q on error", cacheStatus, QueryEmbeddingCacheStatusNotApplicable)
	}
	if !strings.Contains(err.Error(), "resolve query embedding cache key") {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := embedder.CallCount(); got != 0 {
		t.Fatalf("embedder calls = %d, want 0 when cache key resolution fails", got)
	}
}

func TestSearchDiagnosticsSetQueryEmbeddingCacheStatusKeepsHitBoolInSync(t *testing.T) {
	tests := []struct {
		name   string
		status QueryEmbeddingCacheStatus
		want   bool
	}{
		{name: "not applicable", status: QueryEmbeddingCacheStatusNotApplicable, want: false},
		{name: "disabled", status: QueryEmbeddingCacheStatusDisabled, want: false},
		{name: "bypassed", status: QueryEmbeddingCacheStatusBypassed, want: false},
		{name: "miss", status: QueryEmbeddingCacheStatusMiss, want: false},
		{name: "hit", status: QueryEmbeddingCacheStatusHit, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics := SearchDiagnostics{}
			diagnostics.setQueryEmbeddingCacheStatus(tt.status)

			if diagnostics.QueryEmbeddingCacheStatus != tt.status {
				t.Fatalf("cache status = %q, want %q", diagnostics.QueryEmbeddingCacheStatus, tt.status)
			}
			if diagnostics.QueryEmbeddingCacheHit != tt.want {
				t.Fatalf("cache hit = %v, want %v for status %q", diagnostics.QueryEmbeddingCacheHit, tt.want, tt.status)
			}
		})
	}
}

func TestRestartDocumentWriteReturnsConcurrentErrorThatSupportsErrorsIs(t *testing.T) {
	store := &Store{}
	_, err := store.restartDocumentWrite(context.Background(), UpsertDocumentRequest{
		Collection: "docs",
		DocumentID: "doc-1",
	}, false, 1)
	if err == nil {
		t.Fatal("expected restartDocumentWrite() to fail when retries are exhausted")
	}
	if !errors.Is(err, ErrDocumentChangedConcurrently) {
		t.Fatalf("expected errors.Is(..., ErrDocumentChangedConcurrently), got %v", err)
	}
}

type queryCacheScopeKey struct{}

type queryCacheBypassKey struct{}

type plainCountingEmbedder struct {
	mu         sync.Mutex
	calls      int
	dimensions int
}

func (e *plainCountingEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.calls++
	dimensions := e.dimensions
	e.mu.Unlock()

	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = makeVector(dimensions, i%dimensions)
	}
	return out, nil
}

func (e *plainCountingEmbedder) CallCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

type cacheAwareCountingEmbedder struct {
	mu          sync.Mutex
	calls       int
	dimensions  int
	cacheKeyErr error
}

func (e *cacheAwareCountingEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.calls++
	dimensions := e.dimensions
	e.mu.Unlock()

	scope, _ := ctx.Value(queryCacheScopeKey{}).(string)
	if scope == "" {
		scope = "default"
	}
	hotIndex := scopeHotIndex(scope, dimensions)

	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = makeVector(dimensions, (hotIndex+i)%dimensions)
	}
	return out, nil
}

func (e *cacheAwareCountingEmbedder) QueryEmbeddingCacheKey(ctx context.Context, normalizedQuery string) (string, bool, error) {
	if e.cacheKeyErr != nil {
		return "", false, e.cacheKeyErr
	}
	if bypass, _ := ctx.Value(queryCacheBypassKey{}).(bool); bypass {
		return "", false, nil
	}
	scope, _ := ctx.Value(queryCacheScopeKey{}).(string)
	if scope == "" {
		scope = "default"
	}
	return scope + ":" + normalizedQuery, true, nil
}

func (e *cacheAwareCountingEmbedder) CallCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

func scopeHotIndex(scope string, dimensions int) int {
	var sum int
	for _, r := range scope {
		sum += int(r)
	}
	if dimensions <= 0 {
		return 0
	}
	return sum % dimensions
}
