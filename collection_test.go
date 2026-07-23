package simplykb

import (
	"context"
	"testing"
)

func TestDeleteCollectionRejectsEmptyCollectionWithoutUsingDefault(t *testing.T) {
	store := &Store{
		cfg: Config{DefaultCollection: "must-not-be-deleted"}.normalized(),
	}

	err := store.DeleteCollection(context.Background(), " \t ")
	if err == nil {
		t.Fatal("expected empty collection to be rejected")
	}
}

func TestIntegrationCollectionStatsAndDeleteCollectionAreExact(t *testing.T) {
	ctx := context.Background()
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	store := newIntegrationStore(t, databaseURL, schema, Config{
		DefaultCollection:   "default-docs",
		EmbeddingDimensions: 256,
		Embedder:            NewHashEmbedder(256),
		Splitter: DefaultSplitter{
			ChunkSize: 20,
			Overlap:   0,
		},
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	for _, req := range []UpsertDocumentRequest{
		{Collection: "workspace:one", DocumentID: "doc-1", Content: "one two three four five six seven eight"},
		{Collection: "workspace:one", DocumentID: "doc-2", Content: "short"},
		{Collection: "workspace:two", DocumentID: "doc-3", Content: "must survive exact deletion"},
	} {
		if _, err := store.UpsertDocument(ctx, req); err != nil {
			t.Fatalf("UpsertDocument(%s/%s) error = %v", req.Collection, req.DocumentID, err)
		}
	}

	wantOne := queryCollectionStats(t, ctx, store, "workspace:one")
	gotOne, err := store.CollectionStats(ctx, "workspace:one")
	if err != nil {
		t.Fatalf("CollectionStats(workspace:one) error = %v", err)
	}
	if gotOne != wantOne {
		t.Fatalf("CollectionStats(workspace:one) = %+v, want %+v", gotOne, wantOne)
	}

	missing, err := store.CollectionStats(ctx, "workspace:missing")
	if err != nil {
		t.Fatalf("CollectionStats(missing) error = %v", err)
	}
	if missing.Collection != "workspace:missing" || missing.DocumentCount != 0 || missing.ChunkCount != 0 {
		t.Fatalf("CollectionStats(missing) = %+v, want named zero stats", missing)
	}

	if err := store.DeleteCollection(ctx, "workspace:one"); err != nil {
		t.Fatalf("DeleteCollection(workspace:one) error = %v", err)
	}
	afterDelete, err := store.CollectionStats(ctx, "workspace:one")
	if err != nil {
		t.Fatalf("CollectionStats(after delete) error = %v", err)
	}
	if afterDelete.DocumentCount != 0 || afterDelete.ChunkCount != 0 {
		t.Fatalf("deleted collection stats = %+v, want zero", afterDelete)
	}

	survivor, err := store.CollectionStats(ctx, "workspace:two")
	if err != nil {
		t.Fatalf("CollectionStats(workspace:two) error = %v", err)
	}
	if survivor.DocumentCount != 1 || survivor.ChunkCount == 0 {
		t.Fatalf("neighbor collection changed by exact deletion: %+v", survivor)
	}
}

func queryCollectionStats(t *testing.T, ctx context.Context, store *Store, collection string) CollectionStats {
	t.Helper()
	var stats CollectionStats
	stats.Collection = collection
	if err := store.pool.QueryRow(ctx, `
SELECT
    (SELECT count(*) FROM kb_documents WHERE collection = $1),
    (SELECT count(*) FROM kb_chunks WHERE collection = $1)
`, collection).Scan(&stats.DocumentCount, &stats.ChunkCount); err != nil {
		t.Fatalf("query collection stats: %v", err)
	}
	return stats
}
