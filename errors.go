package simplykb

import (
	"errors"
	"strconv"
)

var (
	ErrDocumentChangedConcurrently = errors.New("document changed concurrently")
	ErrEmptyChunk                  = errors.New("chunk is empty")
	ErrChunkTooLarge               = errors.New("chunk exceeds configured character limit")
)

type ChunkLimitError struct {
	Ordinal     int
	ActualChars int
	MaxChars    int
}

func (e *ChunkLimitError) Error() string {
	return "chunk " + strconv.Itoa(e.Ordinal) + " has " + strconv.Itoa(e.ActualChars) +
		" characters, exceeding configured limit " + strconv.Itoa(e.MaxChars)
}

func (e *ChunkLimitError) Unwrap() error {
	return ErrChunkTooLarge
}
