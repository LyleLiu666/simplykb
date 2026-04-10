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

	appendChunk := func(force bool) {
		if len(current) == 0 {
			return
		}
		if !force && len(current) < chunkSize {
			return
		}
		content := strings.TrimSpace(string(current))
		if content == "" {
			current = nil
			return
		}
		chunks = append(chunks, ChunkDraft{
			Ordinal: len(chunks),
			Content: content,
		})
		tail := tailRunes(current, overlap)
		current = append([]rune{}, tail...)
	}

	for _, paragraph := range paragraphs {
		if paragraph == "" {
			continue
		}
		segment := []rune(paragraph)
		if len(segment) > chunkSize {
			appendChunk(true)
			for len(segment) > chunkSize {
				part := strings.TrimSpace(string(segment[:chunkSize]))
				chunks = append(chunks, ChunkDraft{
					Ordinal: len(chunks),
					Content: part,
				})
				segment = segment[max(0, chunkSize-overlap):]
			}
			if len(segment) > 0 {
				current = append([]rune{}, segment...)
			}
			continue
		}

		if len(current) > 0 {
			current = append(current, []rune("\n\n")...)
		}
		current = append(current, segment...)
		appendChunk(false)
	}

	appendChunk(true)
	if len(chunks) == 0 && len(current) > 0 {
		chunks = append(chunks, ChunkDraft{
			Ordinal: 0,
			Content: strings.TrimSpace(string(current)),
		})
	}
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
