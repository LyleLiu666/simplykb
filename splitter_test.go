package simplykb

import (
	"strings"
	"testing"
)

func TestDefaultSplitterSplit(t *testing.T) {
	splitter := DefaultSplitter{
		ChunkSize: 32,
		Overlap:   8,
	}

	chunks, err := splitter.Split("first block\n\nsecond block with more content\n\nthird block")
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected more than one chunk, got %d", len(chunks))
	}
	if chunks[0].Ordinal != 0 || chunks[1].Ordinal != 1 {
		t.Fatalf("unexpected ordinals: %+v", chunks)
	}
	if !strings.Contains(chunks[1].Content, "content") {
		t.Fatalf("expected overlap chunk to preserve content, got %q", chunks[1].Content)
	}
}

func TestDefaultSplitterRejectsEmptyText(t *testing.T) {
	_, err := NewDefaultSplitter().Split("   ")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}
