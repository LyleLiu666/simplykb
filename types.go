package simplykb

import "context"

type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
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

type ChunkDraft struct {
	Ordinal int
	Content string
}
