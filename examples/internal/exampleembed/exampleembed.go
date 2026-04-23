package exampleembed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/LyleLiu666/simplykb"
)

const (
	ProviderHash             = "hash"
	ProviderOpenAICompatible = "openai_compatible"

	defaultHashDimensions = 256
	defaultTimeoutSeconds = 30

	maxEmbeddingErrorBodyBytes = 1 << 20
)

type Config struct {
	Provider   string
	Dimensions int
	Timeout    time.Duration
	URL        string
	APIKey     string
	Model      string
}

type openAICompatibleEmbedder struct {
	client             *http.Client
	url                string
	apiKey             string
	model              string
	expectedDimensions int
}

type embeddingsRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format,omitempty"`
}

type embeddingsResponse struct {
	Data []embeddingItem `json:"data"`
}

type embeddingItem struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

func LoadConfigFromEnv() (Config, error) {
	return loadConfigFromEnv(ProviderHash)
}

func LoadOpenAICompatibleConfigFromEnv() (Config, error) {
	provider := strings.TrimSpace(os.Getenv("SIMPLYKB_EMBEDDER_PROVIDER"))
	if provider != "" && provider != ProviderOpenAICompatible {
		return Config{}, fmt.Errorf("SIMPLYKB_EMBEDDER_PROVIDER=%q conflicts with the openai_compatible example", provider)
	}
	return loadConfigFromEnv(ProviderOpenAICompatible)
}

func loadConfigFromEnv(defaultProvider string) (Config, error) {
	provider := strings.TrimSpace(os.Getenv("SIMPLYKB_EMBEDDER_PROVIDER"))
	if provider == "" {
		provider = defaultProvider
	}

	timeoutSeconds, err := intEnvOrDefault("SIMPLYKB_EMBEDDING_TIMEOUT_SECONDS", defaultTimeoutSeconds)
	if err != nil {
		return Config{}, err
	}

	dimensionsRaw := strings.TrimSpace(os.Getenv("SIMPLYKB_EMBEDDING_DIMENSIONS"))
	dimensions := 0
	if dimensionsRaw != "" {
		dimensions, err = intEnvOrDefault("SIMPLYKB_EMBEDDING_DIMENSIONS", 0)
		if err != nil {
			return Config{}, err
		}
	}

	cfg := Config{
		Provider: provider,
		Timeout:  time.Duration(timeoutSeconds) * time.Second,
		URL:      strings.TrimSpace(os.Getenv("SIMPLYKB_EMBEDDING_URL")),
		APIKey:   strings.TrimSpace(os.Getenv("SIMPLYKB_EMBEDDING_API_KEY")),
		Model:    strings.TrimSpace(os.Getenv("SIMPLYKB_EMBEDDING_MODEL")),
	}

	switch cfg.Provider {
	case ProviderHash:
		if dimensions <= 0 {
			dimensions = defaultHashDimensions
		}
		cfg.Dimensions = dimensions
	case ProviderOpenAICompatible:
		cfg.Dimensions = dimensions
	default:
		return Config{}, fmt.Errorf("SIMPLYKB_EMBEDDER_PROVIDER must be one of %q or %q", ProviderHash, ProviderOpenAICompatible)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	switch c.Provider {
	case ProviderHash:
		if c.Dimensions <= 0 {
			return errors.New("SIMPLYKB_EMBEDDING_DIMENSIONS must be greater than 0")
		}
		if c.Timeout <= 0 {
			return errors.New("SIMPLYKB_EMBEDDING_TIMEOUT_SECONDS must be greater than 0")
		}
		return nil
	case ProviderOpenAICompatible:
		switch {
		case c.URL == "":
			return errors.New("SIMPLYKB_EMBEDDING_URL is required")
		case c.Model == "":
			return errors.New("SIMPLYKB_EMBEDDING_MODEL is required")
		case c.Dimensions <= 0:
			return errors.New("SIMPLYKB_EMBEDDING_DIMENSIONS must be greater than 0")
		case c.Timeout <= 0:
			return errors.New("SIMPLYKB_EMBEDDING_TIMEOUT_SECONDS must be greater than 0")
		default:
			return nil
		}
	default:
		return fmt.Errorf("unsupported embedder provider %q", c.Provider)
	}
}

func (c Config) NewEmbedder() (simplykb.Embedder, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	switch c.Provider {
	case ProviderHash:
		return simplykb.NewHashEmbedder(c.Dimensions), nil
	case ProviderOpenAICompatible:
		return &openAICompatibleEmbedder{
			client: &http.Client{
				Timeout: c.Timeout,
			},
			url:                c.URL,
			apiKey:             c.APIKey,
			model:              c.Model,
			expectedDimensions: c.Dimensions,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported embedder provider %q", c.Provider)
	}
}

func (e *openAICompatibleEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errors.New("embedder is nil")
	}
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	body, err := json.Marshal(embeddingsRequest{
		Model:          e.model,
		Input:          texts,
		EncodingFormat: "float",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embedding api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxEmbeddingErrorBodyBytes))
		if err != nil {
			return nil, fmt.Errorf("read embedding error response: %w", err)
		}
		return nil, fmt.Errorf("embedding api returned %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	var payload embeddingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(payload.Data) != len(texts) {
		return nil, fmt.Errorf("embedding api returned %d vectors for %d texts", len(payload.Data), len(texts))
	}

	sort.Slice(payload.Data, func(i, j int) bool {
		return payload.Data[i].Index < payload.Data[j].Index
	})

	out := make([][]float32, 0, len(payload.Data))
	for i, item := range payload.Data {
		if item.Index != i {
			return nil, fmt.Errorf("embedding api returned unexpected index %d at position %d", item.Index, i)
		}
		if len(item.Embedding) != e.expectedDimensions {
			return nil, fmt.Errorf("embedding dimensions mismatch: got %d want %d", len(item.Embedding), e.expectedDimensions)
		}
		out = append(out, item.Embedding)
	}

	return out, nil
}

func (e *openAICompatibleEmbedder) QueryEmbeddingCacheKey(ctx context.Context, normalizedQuery string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	return normalizedQuery, true, nil
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
