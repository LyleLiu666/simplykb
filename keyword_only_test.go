package simplykb

import (
	"context"
	"testing"
)

func TestKeywordOnlyNeedsNoEmbedderOrDimensions(t *testing.T) {
	cfg := (Config{DatabaseURL: "postgres://unused", KeywordOnly: true}).normalized()
	if err := cfg.validate(); err != nil {
		t.Fatal(err)
	}
	store := &Store{cfg: cfg}
	prepared, err := store.prepareDocumentWrite(context.Background(), UpsertDocumentRequest{Content: "A genuine keyword document."})
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.chunks) == 0 || len(prepared.vectors) != 0 {
		t.Fatalf("keyword-only write must have chunks and no vectors: %#v", prepared)
	}
	request := store.normalizeSearchRequest(SearchRequest{Query: "keyword"})
	if request.Mode != SearchModeKeyword {
		t.Fatalf("default mode = %q", request.Mode)
	}
	if err := store.validateSearchRequest(SearchRequest{Query: "keyword", Mode: SearchModeHybrid}); err == nil {
		t.Fatal("hybrid search must fail explicitly without embeddings")
	}
}

func TestIntegrationKeywordOnlyCanBackfillVectorsWithoutLosingDocuments(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	lexical := newIntegrationStore(t, databaseURL, schema, Config{KeywordOnly: true})
	if err := lexical.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	request := UpsertDocumentRequest{DocumentID: "persistent", Title: "Scientific writing", Content: "PubMed literature and scientific writing workflows."}
	if _, err := lexical.UpsertDocument(ctx, request); err != nil {
		t.Fatal(err)
	}
	hits, err := lexical.Search(ctx, SearchRequest{Query: "PubMed"})
	if err != nil || len(hits) == 0 {
		t.Fatalf("BM25 search: %#v %v", hits, err)
	}
	var vectors int
	if err := lexical.pool.QueryRow(ctx, "SELECT COUNT(*) FROM kb_chunks WHERE embedding IS NOT NULL").Scan(&vectors); err != nil || vectors != 0 {
		t.Fatalf("unexpected synthetic vectors: %d %v", vectors, err)
	}
	semantic := newIntegrationStore(t, databaseURL, schema, Config{EmbeddingDimensions: 256, Embedder: NewHashEmbedder(256)})
	if err := semantic.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := semantic.UpsertDocument(ctx, request); err != nil {
		t.Fatal(err)
	}
	if err := semantic.pool.QueryRow(ctx, "SELECT COUNT(*) FROM kb_chunks WHERE embedding IS NOT NULL").Scan(&vectors); err != nil || vectors == 0 {
		t.Fatalf("unchanged document was not backfilled: %d %v", vectors, err)
	}
	hits, err = semantic.Search(ctx, SearchRequest{Query: "PubMed", Mode: SearchModeKeyword})
	if err != nil || len(hits) == 0 || hits[0].DocumentID != "persistent" {
		t.Fatalf("keyword document lost during vector enablement: %#v %v", hits, err)
	}
}
