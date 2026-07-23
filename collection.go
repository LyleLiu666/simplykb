package simplykb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Store) CollectionStats(ctx context.Context, collection string) (CollectionStats, error) {
	if err := s.ensureReady(); err != nil {
		return CollectionStats{}, err
	}
	collection = s.resolveCollection(collection)

	stats := CollectionStats{Collection: collection}
	err := s.pool.QueryRow(ctx, `
SELECT
    (SELECT count(*) FROM kb_documents WHERE collection = $1),
    (SELECT count(*) FROM kb_chunks WHERE collection = $1)
`, collection).Scan(&stats.DocumentCount, &stats.ChunkCount)
	if err != nil {
		return CollectionStats{}, fmt.Errorf("read collection stats: %w", err)
	}
	return stats, nil
}

func (s *Store) DeleteCollection(ctx context.Context, collection string) error {
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return errors.New("collection is required")
	}
	if err := s.ensureReady(); err != nil {
		return err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin delete collection tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `
DELETE FROM kb_documents
WHERE collection = $1
`, collection); err != nil {
		return fmt.Errorf("delete collection documents: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete collection: %w", err)
	}
	return nil
}
