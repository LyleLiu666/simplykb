package simplykb

import (
	"strings"
	"testing"
	"unicode/utf8"
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

func TestDefaultSplitterNeverExceedsConfiguredTargetWhenJoiningParagraphs(t *testing.T) {
	splitter := DefaultSplitter{
		ChunkSize: 10,
		Overlap:   2,
	}

	chunks, err := splitter.Split("123456\n\nabcdef")
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected paragraph join to split, got %+v", chunks)
	}
	for _, chunk := range chunks {
		if got := utf8.RuneCountInString(chunk.Content); got > splitter.ChunkSize {
			t.Fatalf("chunk %d has %d runes, want <= %d: %q", chunk.Ordinal, got, splitter.ChunkSize, chunk.Content)
		}
	}
}

func TestDefaultSplitterUsesUnicodeCharacterCount(t *testing.T) {
	splitter := DefaultSplitter{
		ChunkSize: 5,
		Overlap:   1,
	}

	chunks, err := splitter.Split("知识治理需要边界")
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected unicode input to split, got %+v", chunks)
	}
	for _, chunk := range chunks {
		if got := utf8.RuneCountInString(chunk.Content); got > splitter.ChunkSize {
			t.Fatalf("chunk %d has %d runes, want <= %d: %q", chunk.Ordinal, got, splitter.ChunkSize, chunk.Content)
		}
	}
}

func TestDefaultSplitterDoesNotEmitOverlapOnlyChunkAtParagraphBoundary(t *testing.T) {
	splitter := DefaultSplitter{
		ChunkSize: 10,
		Overlap:   8,
	}

	chunks, err := splitter.Split("abcdefghij\n\nKLMN")
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %+v", chunks)
	}
	if chunks[1].Content == "cdefghij" {
		t.Fatalf("second chunk only repeats overlap without new paragraph content: %+v", chunks)
	}
	if !strings.Contains(strings.Join(chunkContents(chunks), ""), "KL") {
		t.Fatalf("expected next paragraph content to progress immediately, got %+v", chunks)
	}
}

func chunkContents(chunks []ChunkDraft) []string {
	out := make([]string, len(chunks))
	for i := range chunks {
		out[i] = chunks[i].Content
	}
	return out
}
