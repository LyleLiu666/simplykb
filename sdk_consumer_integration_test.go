package simplykb

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationExternalModuleCanUseSDK(t *testing.T) {
	databaseURL := integrationDatabaseURL(t)
	schema := createIntegrationSchema(t, databaseURL)
	databaseURL = databaseURLWithSearchPath(t, databaseURL, schema)

	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("get repo root: %v", err)
	}

	tempDir := t.TempDir()
	goMod := fmt.Sprintf(`module example.com/simplykb-consumer

go 1.25.0

require github.com/LyleLiu666/simplykb v0.0.0

replace github.com/LyleLiu666/simplykb => %s
`, repoRoot)
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	mainProgram := `package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/LyleLiu666/simplykb"
)

func main() {
	ctx := context.Background()
	store, err := simplykb.New(ctx, simplykb.Config{
		DatabaseURL:             os.Getenv("SIMPLYKB_DATABASE_URL"),
		DefaultCollection:       "external",
		EmbeddingDimensions:     256,
		Embedder:                simplykb.NewHashEmbedder(256),
		QueryEmbeddingCacheSize: 8,
	})
	if err != nil {
		log.Fatalf("new store: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	req := simplykb.UpsertDocumentRequest{
		DocumentID: "doc-1",
		Title:      "External Consumer",
		Content:    "embedded sdk recall works from another module",
	}
	if _, err := store.UpsertDocument(ctx, req); err != nil {
		log.Fatalf("upsert: %v", err)
	}
	if _, err := store.ReindexDocument(ctx, req); err != nil {
		log.Fatalf("reindex: %v", err)
	}

	first, err := store.SearchDetailed(ctx, simplykb.SearchRequest{
		Query: "embedded sdk",
		Limit: 1,
		Mode:  simplykb.SearchModeVector,
	})
	if err != nil {
		log.Fatalf("first detailed search: %v", err)
	}
	if len(first.Hits) == 0 {
		log.Fatal("first detailed search returned no hits")
	}
	if first.Diagnostics.QueryEmbeddingCacheHit {
		log.Fatal("first detailed search should miss cache")
	}
	if first.Diagnostics.QueryEmbeddingCacheStatus != simplykb.QueryEmbeddingCacheStatusMiss {
		log.Fatalf("first detailed search cache status: %s", first.Diagnostics.QueryEmbeddingCacheStatus)
	}

	second, err := store.SearchDetailed(ctx, simplykb.SearchRequest{
		Query: "embedded sdk",
		Limit: 1,
		Mode:  simplykb.SearchModeVector,
	})
	if err != nil {
		log.Fatalf("second detailed search: %v", err)
	}
	if !second.Diagnostics.QueryEmbeddingCacheHit {
		log.Fatal("second detailed search should hit cache")
	}
	if second.Diagnostics.QueryEmbeddingCacheStatus != simplykb.QueryEmbeddingCacheStatusHit {
		log.Fatalf("second detailed search cache status: %s", second.Diagnostics.QueryEmbeddingCacheStatus)
	}

	hits, err := store.Search(ctx, simplykb.SearchRequest{
		Query: "embedded sdk",
		Limit: 1,
	})
	if err != nil {
		log.Fatalf("search: %v", err)
	}
	if len(hits) == 0 {
		log.Fatal("no hits returned")
	}
	fmt.Printf("external-consumer-ok %s %s\n", hits[0].DocumentID, second.Diagnostics.QueryEmbeddingCacheStatus)
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(mainProgram), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	tidyCmd.Env = append(os.Environ(), "GOWORK=off")
	tidyOutput, err := tidyCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go mod tidy external consumer: %v\n%s", err, tidyOutput)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "GOWORK=off", "SIMPLYKB_DATABASE_URL="+databaseURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run external consumer: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "external-consumer-ok doc-1 hit") {
		t.Fatalf("unexpected external consumer output: %s", output)
	}
}
