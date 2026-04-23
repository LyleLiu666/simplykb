package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/LyleLiu666/simplykb"
	"github.com/LyleLiu666/simplykb/examples/internal/exampleembed"
	"github.com/LyleLiu666/simplykb/examples/internal/exampleenv"
)

const defaultExtensions = ".html,.htm,.json,.markdown,.md,.text,.txt,.yaml,.yml"

type ingestConfig struct {
	DatabaseURL string
	Collection  string
	SourceDir   string
	Extensions  []string
	SearchQuery string
	SearchLimit int
	StrictMode  bool
}

type sourceDocument struct {
	RelativePath string
	Request      simplykb.UpsertDocumentRequest
}

type skippedSourceDocument struct {
	RelativePath string
	Reason       string
}

func main() {
	ctx := context.Background()

	cfg, err := loadIngestConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	embedCfg, err := exampleembed.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("load embedder config: %v", err)
	}
	embedder, err := embedCfg.NewEmbedder()
	if err != nil {
		log.Fatalf("create embedder: %v", err)
	}

	store, err := simplykb.New(ctx, simplykb.Config{
		DatabaseURL:         cfg.DatabaseURL,
		DefaultCollection:   cfg.Collection,
		EmbeddingDimensions: embedCfg.Dimensions,
		Embedder:            embedder,
	})
	if err != nil {
		log.Fatalf("create store: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate schema: %v", err)
	}

	docs, skipped, err := collectDocuments(cfg.SourceDir, cfg.Extensions, cfg.StrictMode)
	if err != nil {
		log.Fatalf("collect documents: %v", err)
	}
	for _, item := range skipped {
		fmt.Fprintf(os.Stderr, "skipped %s: %s\n", item.RelativePath, item.Reason)
	}
	if len(docs) == 0 {
		log.Fatalf("no ingestable files found under %s for extensions %s", cfg.SourceDir, strings.Join(cfg.Extensions, ","))
	}

	for _, doc := range docs {
		stats, err := store.UpsertDocument(ctx, doc.Request)
		if err != nil {
			log.Fatalf("upsert %s: %v", doc.RelativePath, err)
		}
		fmt.Printf("indexed %s with %d chunks\n", stats.DocumentID, stats.ChunkCount)
	}

	if strings.TrimSpace(cfg.SearchQuery) == "" {
		return
	}

	hits, err := store.Search(ctx, simplykb.SearchRequest{
		Query: cfg.SearchQuery,
		Limit: cfg.SearchLimit,
	})
	if err != nil {
		log.Fatalf("search: %v", err)
	}

	fmt.Println("\nTop hits:")
	for _, hit := range hits {
		fmt.Printf("- %s chunk=%d score=%.4f keyword=%.4f vector=%.4f\n", hit.DocumentID, hit.ChunkNumber, hit.Score, hit.KeywordScore, hit.VectorScore)
		fmt.Printf("  snippet: %s\n", hit.Snippet)
	}
}

func loadIngestConfig() (ingestConfig, error) {
	sourceDir := strings.TrimSpace(os.Getenv("SIMPLYKB_SOURCE_DIR"))
	if sourceDir == "" {
		return ingestConfig{}, errors.New("SIMPLYKB_SOURCE_DIR is required")
	}

	absoluteSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return ingestConfig{}, fmt.Errorf("resolve SIMPLYKB_SOURCE_DIR: %w", err)
	}

	extensions, err := normalizeExtensions(os.Getenv("SIMPLYKB_FILE_EXTENSIONS"))
	if err != nil {
		return ingestConfig{}, err
	}

	searchLimit, err := intEnvOrDefault("SIMPLYKB_SEARCH_LIMIT", 3)
	if err != nil {
		return ingestConfig{}, err
	}
	strictMode, err := boolEnvOrDefault("SIMPLYKB_INGEST_STRICT", false)
	if err != nil {
		return ingestConfig{}, err
	}

	return ingestConfig{
		DatabaseURL: exampleenv.DefaultDatabaseURL(),
		Collection:  exampleenv.StringOrDefault("SIMPLYKB_COLLECTION", "files"),
		SourceDir:   absoluteSourceDir,
		Extensions:  extensions,
		SearchQuery: strings.TrimSpace(os.Getenv("SIMPLYKB_SEARCH_QUERY")),
		SearchLimit: searchLimit,
		StrictMode:  strictMode,
	}, nil
}

func normalizeExtensions(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = defaultExtensions
	}

	seen := make(map[string]struct{})
	var extensions []string
	for _, item := range strings.Split(raw, ",") {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if !strings.HasPrefix(item, ".") {
			item = "." + item
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		extensions = append(extensions, item)
	}
	if len(extensions) == 0 {
		return nil, errors.New("SIMPLYKB_FILE_EXTENSIONS must include at least one extension")
	}
	sort.Strings(extensions)
	return extensions, nil
}

func collectDocuments(root string, extensions []string, strict bool) ([]sourceDocument, []skippedSourceDocument, error) {
	allowed := make(map[string]struct{}, len(extensions))
	for _, ext := range extensions {
		allowed[ext] = struct{}{}
	}

	var docs []sourceDocument
	var skipped []skippedSourceDocument
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if strict {
				return err
			}
			skipped = append(skipped, skippedSourceDocument{
				RelativePath: filepath.ToSlash(path),
				Reason:       err.Error(),
			})
			return nil
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := allowed[ext]; !ok {
			return nil
		}

		doc, err := loadDocument(root, path, ext)
		if err != nil {
			if strict {
				return err
			}
			relativePath, relErr := filepath.Rel(root, path)
			if relErr != nil {
				relativePath = path
			}
			skipped = append(skipped, skippedSourceDocument{
				RelativePath: filepath.ToSlash(relativePath),
				Reason:       err.Error(),
			})
			return nil
		}
		docs = append(docs, doc)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].RelativePath < docs[j].RelativePath
	})
	sort.Slice(skipped, func(i, j int) bool {
		return skipped[i].RelativePath < skipped[j].RelativePath
	})
	return docs, skipped, nil
}

func loadDocument(root string, path string, extension string) (sourceDocument, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return sourceDocument{}, fmt.Errorf("read %s: %w", path, err)
	}
	if !utf8.Valid(bytes) {
		return sourceDocument{}, fmt.Errorf("read %s: file is not valid UTF-8 text", path)
	}
	content := strings.TrimSpace(string(bytes))
	if content == "" {
		return sourceDocument{}, fmt.Errorf("read %s: file is empty after trimming whitespace", path)
	}

	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return sourceDocument{}, fmt.Errorf("resolve relative path for %s: %w", path, err)
	}
	relativePath = filepath.ToSlash(relativePath)

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return sourceDocument{}, fmt.Errorf("resolve absolute path for %s: %w", path, err)
	}
	absolutePath = filepath.ToSlash(absolutePath)

	sourceURL := (&url.URL{
		Scheme: "file",
		Path:   absolutePath,
	}).String()

	baseName := filepath.Base(relativePath)
	title := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	if title == "" {
		title = relativePath
	}

	return sourceDocument{
		RelativePath: relativePath,
		Request: simplykb.UpsertDocumentRequest{
			DocumentID: relativePath,
			Title:      title,
			Content:    content,
			SourceURI:  sourceURL,
			Metadata: map[string]any{
				"path":        relativePath,
				"extension":   extension,
				"source_kind": "file",
			},
		},
	}, nil
}

func intEnvOrDefault(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0", key)
	}
	return value, nil
}

func boolEnvOrDefault(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", key, err)
	}
	return value, nil
}
