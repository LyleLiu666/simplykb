package simplykb

import (
	"errors"
	"fmt"
)

const (
	defaultCollection      = "default"
	defaultSearchLimit     = 5
	defaultCandidateLimit  = 20
	defaultRRFConstant     = 60
	defaultMaxDocumentSize = 2 * 1024 * 1024
)

type Config struct {
	DatabaseURL         string
	DefaultCollection   string
	EmbeddingDimensions int
	Embedder            Embedder
	Splitter            Splitter
	MinConns            int32
	MaxConns            int32
	DefaultSearchLimit  int
	CandidateLimit      int
	RRFConstant         int
	MaxDocumentBytes    int
}

func (c Config) normalized() Config {
	if c.DefaultCollection == "" {
		c.DefaultCollection = defaultCollection
	}
	if c.DefaultSearchLimit <= 0 {
		c.DefaultSearchLimit = defaultSearchLimit
	}
	if c.CandidateLimit <= 0 {
		c.CandidateLimit = defaultCandidateLimit
	}
	if c.RRFConstant <= 0 {
		c.RRFConstant = defaultRRFConstant
	}
	if c.MaxDocumentBytes <= 0 {
		c.MaxDocumentBytes = defaultMaxDocumentSize
	}
	if c.Splitter == nil {
		c.Splitter = NewDefaultSplitter()
	}
	return c
}

func (c Config) validate() error {
	if c.DatabaseURL == "" {
		return errors.New("database url is required")
	}
	if c.EmbeddingDimensions <= 0 {
		return errors.New("embedding dimensions must be greater than 0")
	}
	if c.Embedder == nil {
		return errors.New("embedder is required")
	}
	if c.MinConns < 0 {
		return errors.New("min conns cannot be negative")
	}
	if c.MaxConns < 0 {
		return errors.New("max conns cannot be negative")
	}
	if c.MaxConns > 0 && c.MinConns > c.MaxConns {
		return fmt.Errorf("min conns %d cannot exceed max conns %d", c.MinConns, c.MaxConns)
	}
	if c.CandidateLimit > 0 && c.DefaultSearchLimit > c.CandidateLimit {
		return fmt.Errorf("default search limit %d cannot exceed candidate limit %d", c.DefaultSearchLimit, c.CandidateLimit)
	}
	return nil
}
