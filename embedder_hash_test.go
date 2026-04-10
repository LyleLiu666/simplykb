package simplykb

import (
	"context"
	"math"
	"testing"
)

func TestHashEmbedderIsDeterministic(t *testing.T) {
	embedder := NewHashEmbedder(64)

	vectors, err := embedder.Embed(context.Background(), []string{"BM25 is good", "BM25 is good"})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(vectors) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vectors))
	}
	if cosine(vectors[0], vectors[1]) < 0.999 {
		t.Fatalf("expected deterministic vectors, got cosine=%f", cosine(vectors[0], vectors[1]))
	}
}

func TestHashEmbedderPreservesSomeSimilarity(t *testing.T) {
	embedder := NewHashEmbedder(64)

	vectors, err := embedder.Embed(context.Background(), []string{
		"keyword search and bm25",
		"bm25 keyword retrieval",
		"ocean waves and beaches",
	})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	closeScore := cosine(vectors[0], vectors[1])
	farScore := cosine(vectors[0], vectors[2])
	if closeScore <= farScore {
		t.Fatalf("expected related texts to be closer, got close=%f far=%f", closeScore, farScore)
	}
}

func cosine(a, b []float32) float64 {
	var dot float64
	for i := range a {
		dot += float64(a[i] * b[i])
	}
	return math.Max(-1, math.Min(1, dot))
}
