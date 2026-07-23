package simplykb

import (
	"errors"
	"strings"
	"testing"
)

func TestChunkLimitErrorSupportsErrorsIsAndAs(t *testing.T) {
	err := error(&ChunkLimitError{
		Ordinal:     2,
		ActualChars: 5001,
		MaxChars:    5000,
	})

	if !errors.Is(err, ErrChunkTooLarge) {
		t.Fatalf("errors.Is(err, ErrChunkTooLarge) = false, err = %v", err)
	}
	var limitErr *ChunkLimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("errors.As(err, *ChunkLimitError) = false, err = %v", err)
	}
	if limitErr.Ordinal != 2 || limitErr.ActualChars != 5001 || limitErr.MaxChars != 5000 {
		t.Fatalf("unexpected ChunkLimitError: %+v", limitErr)
	}
	for _, value := range []string{"2", "5001", "5000"} {
		if !strings.Contains(err.Error(), value) {
			t.Fatalf("error %q does not contain %q", err, value)
		}
	}
}
