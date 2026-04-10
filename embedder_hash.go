package simplykb

import (
	"context"
	"errors"
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

type HashEmbedder struct {
	dimensions int
}

func NewHashEmbedder(dimensions int) *HashEmbedder {
	return &HashEmbedder{dimensions: dimensions}
}

func (e *HashEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e == nil || e.dimensions <= 0 {
		return nil, errors.New("hash embedder dimensions must be greater than 0")
	}

	out := make([][]float32, 0, len(texts))
	for _, text := range texts {
		vector := make([]float32, e.dimensions)
		for _, token := range collectFeatures(text) {
			index := hashToken(token) % uint32(e.dimensions)
			vector[index] += 1
		}
		normalizeVector(vector)
		out = append(out, vector)
	}
	return out, nil
}

func collectFeatures(text string) []string {
	cleaned := strings.ToLower(strings.TrimSpace(text))
	if cleaned == "" {
		return []string{"<empty>"}
	}

	var features []string
	var compact []rune
	fields := strings.FieldsFunc(cleaned, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	for _, field := range fields {
		if field == "" {
			continue
		}
		features = append(features, field)
	}
	for _, r := range []rune(cleaned) {
		if unicode.IsSpace(r) {
			continue
		}
		compact = append(compact, r)
		features = append(features, string(r))
	}
	for i := 0; i+1 < len(compact); i++ {
		features = append(features, string(compact[i:i+2]))
	}
	return features
}

func hashToken(token string) uint32 {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(token))
	return hasher.Sum32()
}

func normalizeVector(vector []float32) {
	var sum float64
	for _, value := range vector {
		sum += float64(value * value)
	}
	if sum == 0 {
		return
	}
	norm := float32(math.Sqrt(sum))
	for i := range vector {
		vector[i] /= norm
	}
}
