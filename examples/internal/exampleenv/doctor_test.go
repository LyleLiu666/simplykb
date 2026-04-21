package exampleenv

import (
	"context"
	"strings"
	"testing"

	"github.com/LyleLiu666/simplykb"
	"github.com/LyleLiu666/simplykb/internal/sdkmeta"
	"github.com/LyleLiu666/simplykb/internal/testdb"
)

func TestIntegrationCollectDatabaseDiagnostics(t *testing.T) {
	ctx := context.Background()
	databaseURL := testdb.DatabaseURL(t)
	schema := testdb.CreateSchema(t, databaseURL, "simplykb_doctor_test")

	store, err := simplykb.New(ctx, simplykb.Config{
		DatabaseURL:         testdb.URLWithSearchPath(t, databaseURL, schema),
		DefaultCollection:   "integration",
		EmbeddingDimensions: 256,
		Embedder:            simplykb.NewHashEmbedder(256),
	})
	if err != nil {
		t.Fatalf("simplykb.New() error = %v", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := store.UpsertDocument(ctx, simplykb.UpsertDocumentRequest{
		DocumentID: "doc-1",
		Title:      "Diagnostics",
		Content:    "doctor mode should show one document and one chunk after indexing.",
	}); err != nil {
		t.Fatalf("UpsertDocument() error = %v", err)
	}

	report, err := CollectDatabaseDiagnostics(ctx, testdb.URLWithSearchPath(t, databaseURL, schema))
	if err != nil {
		t.Fatalf("CollectDatabaseDiagnostics() error = %v", err)
	}

	if report.CurrentSchema != schema {
		t.Fatalf("CurrentSchema = %q, want %q", report.CurrentSchema, schema)
	}
	if !strings.Contains(report.SearchPath, schema) {
		t.Fatalf("SearchPath = %q, want it to include %q", report.SearchPath, schema)
	}
	if !report.RequiredExtensions["pg_search"] {
		t.Fatal("expected pg_search extension support")
	}
	if !report.RequiredExtensions["vector"] {
		t.Fatal("expected vector extension support")
	}
	if len(report.MissingExtensions) != 0 {
		t.Fatalf("MissingExtensions = %v, want none", report.MissingExtensions)
	}
	if !report.MigrationsTableExists {
		t.Fatal("expected migrations table to exist")
	}
	if len(report.AppliedMigrationVersions) == 0 {
		t.Fatal("expected applied migration versions")
	}
	if report.LatestMigrationVersion == 0 {
		t.Fatal("expected latest migration version to be recorded")
	}
	if report.ExpectedMigrationVersion != sdkmeta.LatestSchemaMigrationVersion {
		t.Fatalf("ExpectedMigrationVersion = %d, want %d", report.ExpectedMigrationVersion, sdkmeta.LatestSchemaMigrationVersion)
	}
	if report.Status() != "ready" {
		t.Fatalf("Status() = %q, want ready", report.Status())
	}
	if report.EmbeddingColumnType != "vector(256)" {
		t.Fatalf("EmbeddingColumnType = %q, want vector(256)", report.EmbeddingColumnType)
	}
	if report.DocumentCount != 1 {
		t.Fatalf("DocumentCount = %d, want 1", report.DocumentCount)
	}
	if report.ChunkCount != 1 {
		t.Fatalf("ChunkCount = %d, want 1", report.ChunkCount)
	}
}

func TestDatabaseDiagnosticsStatusDetectsOutdatedSchema(t *testing.T) {
	report := DatabaseDiagnostics{
		MigrationsTableExists:    true,
		LatestMigrationVersion:   sdkmeta.MigrationVersionIndexes,
		ExpectedMigrationVersion: sdkmeta.LatestSchemaMigrationVersion,
	}

	status := report.Status()
	if !strings.Contains(status, "behind current version") {
		t.Fatalf("Status() = %q, want outdated schema guidance", status)
	}
	if !strings.Contains(status, "2 < 4") {
		t.Fatalf("Status() = %q, want concrete version numbers", status)
	}
}
