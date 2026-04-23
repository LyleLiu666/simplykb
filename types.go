package simplykb

import (
	"context"
	"time"
)

type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// QueryEmbeddingCacheKeyer lets an embedder opt into query embedding caching
// and decide whether request-scoped context must change the cache key or bypass
// caching for one search.
type QueryEmbeddingCacheKeyer interface {
	QueryEmbeddingCacheKey(ctx context.Context, normalizedQuery string) (key string, ok bool, err error)
}

type Splitter interface {
	Split(text string) ([]ChunkDraft, error)
}

type SearchMode string

const (
	SearchModeHybrid  SearchMode = "hybrid"
	SearchModeKeyword SearchMode = "keyword"
	SearchModeVector  SearchMode = "vector"
)

type QueryEmbeddingCacheStatus string

const (
	QueryEmbeddingCacheStatusNotApplicable QueryEmbeddingCacheStatus = "not_applicable"
	QueryEmbeddingCacheStatusDisabled      QueryEmbeddingCacheStatus = "disabled"
	QueryEmbeddingCacheStatusBypassed      QueryEmbeddingCacheStatus = "bypassed"
	QueryEmbeddingCacheStatusMiss          QueryEmbeddingCacheStatus = "miss"
	QueryEmbeddingCacheStatusHit           QueryEmbeddingCacheStatus = "hit"
)

type UpsertDocumentRequest struct {
	Collection string
	DocumentID string
	Title      string
	Content    string
	SourceURI  string
	Tags       []string
	Metadata   map[string]any
}

type DocumentStats struct {
	Collection  string
	DocumentID  string
	ContentHash string
	ChunkCount  int
}

type SearchRequest struct {
	Collection     string
	Query          string
	Limit          int
	CandidateLimit int
	Mode           SearchMode
	MetadataFilter map[string]any
}

type SearchHit struct {
	Collection   string
	DocumentID   string
	ChunkID      string
	ChunkNumber  int
	Title        string
	Content      string
	Snippet      string
	SourceURI    string
	Tags         []string
	Metadata     map[string]any
	Score        float64
	KeywordScore float64
	VectorScore  float64
}

type SearchResponse struct {
	Hits        []SearchHit
	Diagnostics SearchDiagnostics
}

type SearchDiagnostics struct {
	Mode                      SearchMode
	TotalDuration             time.Duration
	KeywordDuration           time.Duration
	VectorDuration            time.Duration
	KeywordCandidateCount     int
	VectorCandidateCount      int
	FusedCandidateCount       int
	QueryEmbeddingCacheStatus QueryEmbeddingCacheStatus
	QueryEmbeddingCacheHit    bool
	HadContextDeadline        bool
}

func (d *SearchDiagnostics) setQueryEmbeddingCacheStatus(status QueryEmbeddingCacheStatus) {
	d.QueryEmbeddingCacheStatus = status
	d.QueryEmbeddingCacheHit = status == QueryEmbeddingCacheStatusHit
}

type ChunkDraft struct {
	Ordinal int
	Content string
}
