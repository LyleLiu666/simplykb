package simplykb

import (
	"errors"
	"strings"
)

const (
	defaultChunkSize = 900
	defaultOverlap   = 120
)

type DefaultSplitter struct {
	ChunkSize int
	Overlap   int
}

func NewDefaultSplitter() DefaultSplitter {
	return DefaultSplitter{
		ChunkSize: defaultChunkSize,
		Overlap:   defaultOverlap,
	}
}

func (s DefaultSplitter) Split(text string) ([]ChunkDraft, error) {
	normalized := normalizeText(text)
	if normalized == "" {
		return nil, errors.New("text is empty")
	}

	chunkSize := s.ChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}
	overlap := s.Overlap
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 4
	}

	paragraphs := splitParagraphs(normalized)
	var chunks []ChunkDraft
	var current []rune
	hasNewContent := false

	emitChunk := func() {
		if len(current) == 0 || !hasNewContent {
			return
		}
		content := strings.TrimSpace(string(current))
		if content == "" {
			current = nil
			hasNewContent = false
			return
		}
		chunks = append(chunks, ChunkDraft{
			Ordinal: len(chunks),
			Content: content,
		})
		tail := tailRunes(current, overlap)
		current = append([]rune{}, tail...)
		hasNewContent = false
	}

	appendRunes := func(input []rune) {
		for len(input) > 0 {
			available := chunkSize - len(current)
			if available == 0 {
				emitChunk()
				available = chunkSize - len(current)
			}
			take := min(available, len(input))
			current = append(current, input[:take]...)
			input = input[take:]
			hasNewContent = true
			if len(current) == chunkSize {
				emitChunk()
			}
		}
	}

	for i, paragraph := range paragraphs {
		if i > 0 && chunkSize-len(current) > len([]rune("\n\n")) {
			appendRunes([]rune("\n\n"))
		}
		appendRunes([]rune(paragraph))
	}

	emitChunk()
	return chunks, nil
}

func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.TrimSpace(text)
}

func splitParagraphs(text string) []string {
	parts := strings.Split(text, "\n\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func tailRunes(input []rune, size int) []rune {
	if size <= 0 || len(input) == 0 {
		return nil
	}
	if len(input) <= size {
		return append([]rune{}, input...)
	}
	return append([]rune{}, input[len(input)-size:]...)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
