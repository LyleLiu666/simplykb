package main

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadIngestConfigRequiresSourceDir(t *testing.T) {
	t.Setenv("SIMPLYKB_SOURCE_DIR", "")

	_, err := loadIngestConfig()
	if err == nil {
		t.Fatal("expected missing source dir to fail")
	}
}

func TestLoadIngestConfigDefaults(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SIMPLYKB_SOURCE_DIR", root)
	t.Setenv("SIMPLYKB_COLLECTION", "")
	t.Setenv("SIMPLYKB_FILE_EXTENSIONS", "")
	t.Setenv("SIMPLYKB_SEARCH_QUERY", "")
	t.Setenv("SIMPLYKB_SEARCH_LIMIT", "")

	cfg, err := loadIngestConfig()
	if err != nil {
		t.Fatalf("loadIngestConfig() error = %v", err)
	}
	if cfg.Collection != "files" {
		t.Fatalf("Collection = %q, want %q", cfg.Collection, "files")
	}
	if cfg.SearchLimit != 3 {
		t.Fatalf("SearchLimit = %d, want 3", cfg.SearchLimit)
	}
	if cfg.StrictMode {
		t.Fatal("StrictMode = true, want false by default")
	}
	if len(cfg.Extensions) == 0 {
		t.Fatal("expected default extensions")
	}
}

func TestLoadIngestConfigParsesStrictMode(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SIMPLYKB_SOURCE_DIR", root)
	t.Setenv("SIMPLYKB_INGEST_STRICT", "true")

	cfg, err := loadIngestConfig()
	if err != nil {
		t.Fatalf("loadIngestConfig() error = %v", err)
	}
	if !cfg.StrictMode {
		t.Fatal("StrictMode = false, want true")
	}
}

func TestNormalizeExtensions(t *testing.T) {
	got, err := normalizeExtensions(".md, TXT ,json,.txt")
	if err != nil {
		t.Fatalf("normalizeExtensions() error = %v", err)
	}

	want := []string{".json", ".md", ".txt"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("extensions[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCollectDocumentsLoadsSupportedTextFiles(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "notes.md"), "hello markdown")
	mustWriteFile(t, filepath.Join(root, "nested", "guide.txt"), "hello text")
	mustWriteFile(t, filepath.Join(root, "image.png"), "not used")

	docs, skipped, err := collectDocuments(root, []string{".md", ".txt"}, false)
	if err != nil {
		t.Fatalf("collectDocuments() error = %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("len(skipped) = %d, want 0", len(skipped))
	}
	if len(docs) != 2 {
		t.Fatalf("len(docs) = %d, want 2", len(docs))
	}

	if docs[0].Request.DocumentID != "nested/guide.txt" {
		t.Fatalf("docs[0].DocumentID = %q", docs[0].Request.DocumentID)
	}
	if docs[1].Request.DocumentID != "notes.md" {
		t.Fatalf("docs[1].DocumentID = %q", docs[1].Request.DocumentID)
	}
	if docs[0].Request.Content != "hello text" {
		t.Fatalf("docs[0].Content = %q", docs[0].Request.Content)
	}
	if docs[1].Request.Metadata["extension"] != ".md" {
		t.Fatalf("docs[1].Metadata[extension] = %#v", docs[1].Request.Metadata["extension"])
	}

	uri, err := url.Parse(docs[0].Request.SourceURI)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if uri.Scheme != "file" {
		t.Fatalf("source uri scheme = %q, want file", uri.Scheme)
	}
}

func TestCollectDocumentsSkipsInvalidUTF8ByDefault(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "good.txt"), "hello")
	path := filepath.Join(root, "broken.txt")
	if err := os.WriteFile(path, []byte{0xff, 0xfe}, 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	docs, skipped, err := collectDocuments(root, []string{".txt"}, false)
	if err != nil {
		t.Fatalf("collectDocuments() error = %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("len(docs) = %d, want 1", len(docs))
	}
	if len(skipped) != 1 {
		t.Fatalf("len(skipped) = %d, want 1", len(skipped))
	}
	if skipped[0].RelativePath != "broken.txt" {
		t.Fatalf("skipped path = %q, want %q", skipped[0].RelativePath, "broken.txt")
	}
	if !strings.Contains(skipped[0].Reason, "not valid UTF-8") {
		t.Fatalf("unexpected skip reason: %q", skipped[0].Reason)
	}
}

func TestCollectDocumentsRejectsInvalidUTF8InStrictMode(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "broken.txt")
	if err := os.WriteFile(path, []byte{0xff, 0xfe}, 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, _, err := collectDocuments(root, []string{".txt"}, true)
	if err == nil {
		t.Fatal("expected invalid utf-8 to fail in strict mode")
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
}
