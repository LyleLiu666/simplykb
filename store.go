package simplykb

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/LyleLiu666/simplykb/internal/sdkmeta"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool       *pgxpool.Pool
	cfg        Config
	queryCache *queryEmbeddingCache
}

type candidateHit struct {
	internalID int64
	hit        SearchHit
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type documentWriteAction int

const (
	documentWriteActionNoop documentWriteAction = iota
	documentWriteActionMetadataRefresh
	documentWriteActionReindex
)

type documentSnapshot struct {
	exists       bool
	internalID   int64
	title        string
	sourceURI    string
	tags         []string
	metadataJSON string
	contentHash  string
	chunkCount   int
	updatedAt    time.Time
}

type preparedDocumentWrite struct {
	chunks  []ChunkDraft
	vectors [][]float32
}

func New(ctx context.Context, cfg Config) (*Store, error) {
	cfg = cfg.normalized()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	if cfg.MinConns > 0 {
		poolConfig.MinConns = cfg.MinConns
	}
	if cfg.MaxConns > 0 {
		poolConfig.MaxConns = cfg.MaxConns
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Store{
		pool:       pool,
		cfg:        cfg,
		queryCache: newQueryEmbeddingCache(cfg.QueryEmbeddingCacheSize),
	}, nil
}

func (s *Store) Close() {
	if s == nil || s.pool == nil {
		return
	}
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	return s.pool.Ping(ctx)
}

func (s *Store) Migrate(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := ensureRequiredExtensions(ctx, tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(73120410)); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	if _, err := tx.Exec(ctx, bootstrapMigrationSQL()); err != nil {
		return fmt.Errorf("bootstrap migrations table: %w", err)
	}

	applied := make(map[int64]struct{})
	rows, err := tx.Query(ctx, `SELECT version FROM kb_schema_migrations`)
	if err != nil {
		return fmt.Errorf("load applied migrations: %w", err)
	}
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			rows.Close()
			return fmt.Errorf("scan migration version: %w", err)
		}
		applied[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate migration versions: %w", err)
	}
	rows.Close()

	for _, migration := range schemaMigrations(s.cfg.EmbeddingDimensions) {
		if _, ok := applied[migration.version]; ok {
			continue
		}
		if _, err := tx.Exec(ctx, migration.sql); err != nil {
			return fmt.Errorf("apply migration %d (%s): %w", migration.version, migration.name, err)
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO kb_schema_migrations (version, name)
VALUES ($1, $2)
`, migration.version, migration.name); err != nil {
			return fmt.Errorf("record migration %d (%s): %w", migration.version, migration.name, err)
		}
	}

	if err := ensureEmbeddingDimensions(ctx, tx, s.cfg.EmbeddingDimensions); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}

func (s *Store) EnsureSchema(ctx context.Context) error {
	return s.Migrate(ctx)
}

func (s *Store) UpsertDocument(ctx context.Context, req UpsertDocumentRequest) (DocumentStats, error) {
	return s.writeDocument(ctx, req, false, 0)
}

func (s *Store) ReindexDocument(ctx context.Context, req UpsertDocumentRequest) (DocumentStats, error) {
	return s.writeDocument(ctx, req, true, 0)
}

func (s *Store) writeDocument(ctx context.Context, req UpsertDocumentRequest, forceReindex bool, attempt int) (DocumentStats, error) {
	if err := s.ensureReady(); err != nil {
		return DocumentStats{}, err
	}
	req = s.normalizeDocumentRequest(req)
	if err := s.validateDocumentRequest(req); err != nil {
		return DocumentStats{}, err
	}

	if len(req.Content) > s.cfg.MaxDocumentBytes {
		return DocumentStats{}, fmt.Errorf("content exceeds max document size of %d bytes", s.cfg.MaxDocumentBytes)
	}

	contentHash := hashText(req.Content)
	metadataJSON, err := canonicalMetadataJSON(req.Metadata)
	if err != nil {
		return DocumentStats{}, fmt.Errorf("marshal metadata: %w", err)
	}

	existing, err := loadDocumentSnapshot(ctx, s.pool, req.Collection, req.DocumentID, false)
	if err != nil {
		return DocumentStats{}, err
	}

	action := classifyDocumentWrite(existing, req, contentHash, metadataJSON, forceReindex)
	switch action {
	case documentWriteActionNoop, documentWriteActionMetadataRefresh:
		stats, shouldRestart, err := s.commitCheapDocumentWrite(ctx, req, contentHash, metadataJSON)
		if err != nil {
			return DocumentStats{}, err
		}
		if shouldRestart {
			return s.restartDocumentWrite(ctx, req, forceReindex, attempt)
		}
		return stats, nil
	case documentWriteActionReindex:
	default:
		return DocumentStats{}, fmt.Errorf("unsupported document write action %d", action)
	}

	prepared, err := s.prepareDocumentWrite(ctx, req)
	if err != nil {
		return DocumentStats{}, err
	}

	stats, shouldRestart, err := s.commitReindexDocumentWrite(ctx, req, contentHash, metadataJSON, prepared, existing, forceReindex)
	if err != nil {
		return DocumentStats{}, err
	}
	if shouldRestart {
		return s.restartDocumentWrite(ctx, req, forceReindex, attempt)
	}
	return stats, nil
}

func (s *Store) DeleteDocument(ctx context.Context, collection string, documentID string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	collection = s.resolveCollection(collection)
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return errors.New("document id is required")
	}
	_, err := s.pool.Exec(ctx, `
DELETE FROM kb_documents
WHERE collection = $1 AND external_id = $2
`, collection, documentID)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
	}
	return nil
}

func (s *Store) Search(ctx context.Context, req SearchRequest) ([]SearchHit, error) {
	response, err := s.SearchDetailed(ctx, req)
	if err != nil {
		return nil, err
	}
	return response.Hits, nil
}

func (s *Store) SearchDetailed(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	startedAt := time.Now()
	if err := s.ensureReady(); err != nil {
		return SearchResponse{}, err
	}
	req = s.normalizeSearchRequest(req)
	if err := s.validateSearchRequest(req); err != nil {
		return SearchResponse{}, err
	}
	filterJSON, err := encodeMetadataFilter(req.MetadataFilter)
	if err != nil {
		return SearchResponse{}, err
	}

	response := SearchResponse{}
	response.Diagnostics.Mode = req.Mode
	response.Diagnostics.setQueryEmbeddingCacheStatus(QueryEmbeddingCacheStatusNotApplicable)
	if _, ok := ctx.Deadline(); ok {
		response.Diagnostics.HadContextDeadline = true
	}

	merged := make(map[int64]*SearchHit)
	if req.Mode == SearchModeHybrid || req.Mode == SearchModeKeyword {
		keywordStartedAt := time.Now()
		keywordHits, err := s.searchKeyword(ctx, req, filterJSON)
		if err != nil {
			return SearchResponse{}, err
		}
		response.Diagnostics.KeywordDuration = time.Since(keywordStartedAt)
		response.Diagnostics.KeywordCandidateCount = len(keywordHits)
		for rank, hit := range keywordHits {
			mergedHit := cloneHit(hit.hit)
			mergedHit.Score += reciprocalRank(rank+1, s.cfg.RRFConstant)
			mergedHit.KeywordScore = hit.hit.KeywordScore
			merged[hit.internalID] = mergedHit
		}
	}

	if req.Mode == SearchModeHybrid || req.Mode == SearchModeVector {
		vectorStartedAt := time.Now()
		queryVector, cacheStatus, err := s.searchQueryVector(ctx, req.Query)
		if err != nil {
			return SearchResponse{}, err
		}
		response.Diagnostics.setQueryEmbeddingCacheStatus(cacheStatus)

		vectorHits, err := s.searchVector(ctx, req, queryVector, filterJSON)
		if err != nil {
			return SearchResponse{}, err
		}
		response.Diagnostics.VectorDuration = time.Since(vectorStartedAt)
		response.Diagnostics.VectorCandidateCount = len(vectorHits)
		for rank, hit := range vectorHits {
			if existing, ok := merged[hit.internalID]; ok {
				existing.Score += reciprocalRank(rank+1, s.cfg.RRFConstant)
				existing.VectorScore = hit.hit.VectorScore
				if existing.Snippet == "" {
					existing.Snippet = hit.hit.Snippet
				}
				continue
			}
			mergedHit := cloneHit(hit.hit)
			mergedHit.Score += reciprocalRank(rank+1, s.cfg.RRFConstant)
			mergedHit.VectorScore = hit.hit.VectorScore
			merged[hit.internalID] = mergedHit
		}
	}

	results := make([]SearchHit, 0, len(merged))
	for _, hit := range merged {
		results = append(results, *hit)
	}
	response.Diagnostics.FusedCandidateCount = len(results)
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			if results[i].DocumentID == results[j].DocumentID {
				return results[i].ChunkNumber < results[j].ChunkNumber
			}
			return results[i].DocumentID < results[j].DocumentID
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}
	response.Hits = results
	response.Diagnostics.TotalDuration = time.Since(startedAt)
	return response, nil
}

func (s *Store) searchKeyword(ctx context.Context, req SearchRequest, filterJSON string) ([]candidateHit, error) {
	rows, err := s.pool.Query(ctx, `
SELECT
    id,
    collection,
    document_external_id,
    chunk_key,
    chunk_no,
    title,
    content,
    source_uri,
    tags,
    metadata,
    COALESCE(paradedb.snippet(search_text), ''),
    paradedb.score(id)
FROM kb_chunks
WHERE collection = $1
  AND search_text ||| $2
  AND metadata @> $4::jsonb
ORDER BY paradedb.score(id) DESC, id DESC
LIMIT $3
`, req.Collection, req.Query, req.CandidateLimit, filterJSON)
	if err != nil {
		return nil, fmt.Errorf("keyword search: %w", err)
	}
	defer rows.Close()
	return readHits(rows, true)
}

func (s *Store) searchVector(ctx context.Context, req SearchRequest, vector []float32, filterJSON string) ([]candidateHit, error) {
	rows, err := s.pool.Query(ctx, `
SELECT
    id,
    collection,
    document_external_id,
    chunk_key,
    chunk_no,
    title,
    content,
    source_uri,
    tags,
    metadata,
    '',
    (1 - (embedding <=> $2::vector))::double precision AS vector_score
FROM kb_chunks
WHERE collection = $1
  AND metadata @> $3::jsonb
ORDER BY embedding <=> $2::vector ASC, id DESC
LIMIT $4
`, req.Collection, vectorLiteral(vector), filterJSON, req.CandidateLimit)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	defer rows.Close()
	return readHits(rows, false)
}

func readHits(rows pgx.Rows, keyword bool) ([]candidateHit, error) {
	var hits []candidateHit
	for rows.Next() {
		var (
			internalID    int64
			hit           SearchHit
			metadataBytes []byte
			score         float64
		)
		if err := rows.Scan(
			&internalID,
			&hit.Collection,
			&hit.DocumentID,
			&hit.ChunkID,
			&hit.ChunkNumber,
			&hit.Title,
			&hit.Content,
			&hit.SourceURI,
			&hit.Tags,
			&metadataBytes,
			&hit.Snippet,
			&score,
		); err != nil {
			return nil, fmt.Errorf("scan hit: %w", err)
		}
		if len(metadataBytes) > 0 {
			if err := json.Unmarshal(metadataBytes, &hit.Metadata); err != nil {
				return nil, fmt.Errorf("decode metadata: %w", err)
			}
		}
		if hit.Metadata == nil {
			hit.Metadata = map[string]any{}
		}
		if keyword {
			hit.KeywordScore = score
		} else {
			hit.VectorScore = score
		}
		hits = append(hits, candidateHit{
			internalID: internalID,
			hit:        hit,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hits, nil
}

func cloneHit(hit SearchHit) *SearchHit {
	metadata := make(map[string]any, len(hit.Metadata))
	for key, value := range hit.Metadata {
		metadata[key] = value
	}
	hit.Metadata = metadata
	return &hit
}

func (s *Store) searchQueryVector(ctx context.Context, normalizedQuery string) ([]float32, QueryEmbeddingCacheStatus, error) {
	cacheKey, useCache, fallbackStatus, err := s.resolveQueryEmbeddingCacheKey(ctx, normalizedQuery)
	if err != nil {
		return nil, QueryEmbeddingCacheStatusNotApplicable, err
	}
	if useCache {
		if cachedVector, ok := s.queryCache.Get(cacheKey); ok {
			return cachedVector, QueryEmbeddingCacheStatusHit, nil
		}
	}

	queryVector, err := s.cfg.Embedder.Embed(ctx, []string{normalizedQuery})
	if err != nil {
		return nil, QueryEmbeddingCacheStatusNotApplicable, fmt.Errorf("embed query: %w", err)
	}
	if len(queryVector) != 1 {
		return nil, QueryEmbeddingCacheStatusNotApplicable, fmt.Errorf("embedder returned %d query vectors", len(queryVector))
	}
	if len(queryVector[0]) != s.cfg.EmbeddingDimensions {
		return nil, QueryEmbeddingCacheStatusNotApplicable, fmt.Errorf("query vector dimension mismatch: got %d want %d", len(queryVector[0]), s.cfg.EmbeddingDimensions)
	}

	if useCache {
		s.queryCache.Put(cacheKey, queryVector[0])
		return cloneVector(queryVector[0]), QueryEmbeddingCacheStatusMiss, nil
	}
	return cloneVector(queryVector[0]), fallbackStatus, nil
}

func (s *Store) resolveQueryEmbeddingCacheKey(ctx context.Context, normalizedQuery string) (string, bool, QueryEmbeddingCacheStatus, error) {
	if s.queryCache == nil {
		return "", false, QueryEmbeddingCacheStatusDisabled, nil
	}

	keyer, ok := s.cfg.Embedder.(QueryEmbeddingCacheKeyer)
	if !ok {
		return "", false, QueryEmbeddingCacheStatusNotApplicable, errors.New("query embedding cache requires embedder implementing QueryEmbeddingCacheKeyer")
	}

	key, ok, err := keyer.QueryEmbeddingCacheKey(ctx, normalizedQuery)
	if err != nil {
		return "", false, QueryEmbeddingCacheStatusNotApplicable, fmt.Errorf("resolve query embedding cache key: %w", err)
	}
	if !ok {
		return "", false, QueryEmbeddingCacheStatusBypassed, nil
	}
	if strings.TrimSpace(key) == "" {
		return "", false, QueryEmbeddingCacheStatusNotApplicable, errors.New("resolve query embedding cache key: empty cache key")
	}
	return key, true, QueryEmbeddingCacheStatusMiss, nil
}

func classifyDocumentWrite(existing documentSnapshot, req UpsertDocumentRequest, contentHash string, metadataJSON string, forceReindex bool) documentWriteAction {
	if forceReindex {
		return documentWriteActionReindex
	}
	if !existing.exists || existing.chunkCount <= 0 {
		return documentWriteActionReindex
	}
	if existing.contentHash != contentHash {
		return documentWriteActionReindex
	}
	if existing.title != req.Title || existing.sourceURI != req.SourceURI || !slices.Equal(existing.tags, req.Tags) || existing.metadataJSON != metadataJSON {
		return documentWriteActionMetadataRefresh
	}
	return documentWriteActionNoop
}

func (s *Store) commitCheapDocumentWrite(ctx context.Context, req UpsertDocumentRequest, contentHash string, metadataJSON string) (DocumentStats, bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return DocumentStats{}, false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := acquireDocumentWriteLock(ctx, tx, req.Collection, req.DocumentID); err != nil {
		return DocumentStats{}, false, err
	}

	locked, err := loadDocumentSnapshot(ctx, tx, req.Collection, req.DocumentID, true)
	if err != nil {
		return DocumentStats{}, false, err
	}

	switch classifyDocumentWrite(locked, req, contentHash, metadataJSON, false) {
	case documentWriteActionNoop:
		if err := tx.Commit(ctx); err != nil {
			return DocumentStats{}, false, fmt.Errorf("commit: %w", err)
		}
		return newDocumentStats(req, contentHash, locked.chunkCount), false, nil
	case documentWriteActionMetadataRefresh:
		if err := s.refreshDocumentMetadata(ctx, tx, locked, req, metadataJSON); err != nil {
			return DocumentStats{}, false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return DocumentStats{}, false, fmt.Errorf("commit: %w", err)
		}
		return newDocumentStats(req, contentHash, locked.chunkCount), false, nil
	case documentWriteActionReindex:
		return DocumentStats{}, true, nil
	default:
		return DocumentStats{}, false, errors.New("unknown cheap document write action")
	}
}

func (s *Store) prepareDocumentWrite(ctx context.Context, req UpsertDocumentRequest) (preparedDocumentWrite, error) {
	chunks, err := s.cfg.Splitter.Split(req.Content)
	if err != nil {
		return preparedDocumentWrite{}, fmt.Errorf("split content: %w", err)
	}
	if len(chunks) == 0 {
		return preparedDocumentWrite{}, errors.New("splitter returned no chunks")
	}

	texts := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		texts = append(texts, chunk.Content)
	}

	vectors, err := s.cfg.Embedder.Embed(ctx, texts)
	if err != nil {
		return preparedDocumentWrite{}, fmt.Errorf("embed chunks: %w", err)
	}
	if len(vectors) != len(chunks) {
		return preparedDocumentWrite{}, fmt.Errorf("embedder returned %d vectors for %d chunks", len(vectors), len(chunks))
	}
	for i := range chunks {
		if len(vectors[i]) != s.cfg.EmbeddingDimensions {
			return preparedDocumentWrite{}, fmt.Errorf("chunk %d vector dimension mismatch: got %d want %d", i, len(vectors[i]), s.cfg.EmbeddingDimensions)
		}
	}

	return preparedDocumentWrite{
		chunks:  chunks,
		vectors: vectors,
	}, nil
}

func (s *Store) commitReindexDocumentWrite(ctx context.Context, req UpsertDocumentRequest, contentHash string, metadataJSON string, prepared preparedDocumentWrite, expected documentSnapshot, forceReindex bool) (DocumentStats, bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return DocumentStats{}, false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := acquireDocumentWriteLock(ctx, tx, req.Collection, req.DocumentID); err != nil {
		return DocumentStats{}, false, err
	}

	locked, err := loadDocumentSnapshot(ctx, tx, req.Collection, req.DocumentID, true)
	if err != nil {
		return DocumentStats{}, false, err
	}
	if !documentSnapshotsEqual(locked, expected) {
		return DocumentStats{}, true, nil
	}

	if !forceReindex {
		switch classifyDocumentWrite(locked, req, contentHash, metadataJSON, false) {
		case documentWriteActionNoop:
			if err := tx.Commit(ctx); err != nil {
				return DocumentStats{}, false, fmt.Errorf("commit: %w", err)
			}
			return newDocumentStats(req, contentHash, locked.chunkCount), false, nil
		case documentWriteActionMetadataRefresh:
			if err := s.refreshDocumentMetadata(ctx, tx, locked, req, metadataJSON); err != nil {
				return DocumentStats{}, false, err
			}
			if err := tx.Commit(ctx); err != nil {
				return DocumentStats{}, false, fmt.Errorf("commit: %w", err)
			}
			return newDocumentStats(req, contentHash, locked.chunkCount), false, nil
		case documentWriteActionReindex:
		default:
			return DocumentStats{}, false, errors.New("unknown reindex document write action")
		}
	}

	internalDocumentID, err := s.upsertDocumentRow(ctx, tx, req, metadataJSON, contentHash)
	if err != nil {
		return DocumentStats{}, false, err
	}
	if err := s.replaceDocumentChunks(ctx, tx, req, internalDocumentID, metadataJSON, prepared); err != nil {
		return DocumentStats{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DocumentStats{}, false, fmt.Errorf("commit: %w", err)
	}

	return newDocumentStats(req, contentHash, len(prepared.chunks)), false, nil
}

func (s *Store) refreshDocumentMetadata(ctx context.Context, tx pgx.Tx, snapshot documentSnapshot, req UpsertDocumentRequest, metadataJSON string) error {
	if _, err := tx.Exec(ctx, `
UPDATE kb_documents
SET title = $3,
    source_uri = $4,
    tags = $5,
    metadata = $6::jsonb,
    updated_at = NOW()
WHERE collection = $1 AND id = $2
`, req.Collection, snapshot.internalID, req.Title, req.SourceURI, req.Tags, metadataJSON); err != nil {
		return fmt.Errorf("update document metadata: %w", err)
	}

	if _, err := tx.Exec(ctx, `
UPDATE kb_chunks
SET title = $3,
    source_uri = $4,
    tags = $5,
    metadata = $6::jsonb,
    updated_at = NOW()
WHERE collection = $1 AND document_id = $2
`, req.Collection, snapshot.internalID, req.Title, req.SourceURI, req.Tags, metadataJSON); err != nil {
		return fmt.Errorf("update chunk metadata: %w", err)
	}

	return nil
}

func (s *Store) upsertDocumentRow(ctx context.Context, tx pgx.Tx, req UpsertDocumentRequest, metadataJSON string, contentHash string) (int64, error) {
	var internalDocumentID int64
	err := tx.QueryRow(ctx, `
INSERT INTO kb_documents (
    collection,
    external_id,
    title,
    source_uri,
    tags,
    metadata,
    content_hash,
    chunk_count,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, 0, NOW())
ON CONFLICT (collection, external_id)
DO UPDATE SET
    title = EXCLUDED.title,
    source_uri = EXCLUDED.source_uri,
    tags = EXCLUDED.tags,
    metadata = EXCLUDED.metadata,
    content_hash = EXCLUDED.content_hash,
    updated_at = NOW()
RETURNING id
`, req.Collection, req.DocumentID, req.Title, req.SourceURI, req.Tags, metadataJSON, contentHash).Scan(&internalDocumentID)
	if err != nil {
		return 0, fmt.Errorf("upsert document: %w", err)
	}
	return internalDocumentID, nil
}

func (s *Store) replaceDocumentChunks(ctx context.Context, tx pgx.Tx, req UpsertDocumentRequest, internalDocumentID int64, metadataJSON string, prepared preparedDocumentWrite) error {
	if _, err := tx.Exec(ctx, `
DELETE FROM kb_chunks
WHERE collection = $1 AND document_id = $2
`, req.Collection, internalDocumentID); err != nil {
		return fmt.Errorf("delete old chunks: %w", err)
	}

	batch := &pgx.Batch{}
	for i, chunk := range prepared.chunks {
		batch.Queue(`
INSERT INTO kb_chunks (
    collection,
    document_id,
    document_external_id,
    chunk_key,
    chunk_no,
    title,
    content,
    source_uri,
    tags,
    metadata,
    embedding
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::vector)
`, req.Collection, internalDocumentID, req.DocumentID, chunkKey(req.DocumentID, chunk.Ordinal), chunk.Ordinal, req.Title, chunk.Content, req.SourceURI, req.Tags, metadataJSON, vectorLiteral(prepared.vectors[i]))
	}

	results := tx.SendBatch(ctx, batch)
	for range prepared.chunks {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return fmt.Errorf("insert chunks: %w", err)
		}
	}
	if err := results.Close(); err != nil {
		return fmt.Errorf("close insert batch: %w", err)
	}

	if _, err := tx.Exec(ctx, `
UPDATE kb_documents
SET chunk_count = $3, updated_at = NOW()
WHERE collection = $1 AND id = $2
`, req.Collection, internalDocumentID, len(prepared.chunks)); err != nil {
		return fmt.Errorf("update chunk count: %w", err)
	}

	return nil
}

func (s *Store) restartDocumentWrite(ctx context.Context, req UpsertDocumentRequest, forceReindex bool, attempt int) (DocumentStats, error) {
	if attempt >= 1 {
		return DocumentStats{}, fmt.Errorf("%w: collection=%s document_id=%s", ErrDocumentChangedConcurrently, req.Collection, req.DocumentID)
	}
	return s.writeDocument(ctx, req, forceReindex, attempt+1)
}

func acquireDocumentWriteLock(ctx context.Context, tx pgx.Tx, collection string, documentID string) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, documentLockKey(collection, documentID)); err != nil {
		return fmt.Errorf("acquire document write lock: %w", err)
	}
	return nil
}

func loadDocumentSnapshot(ctx context.Context, rower queryRower, collection string, documentID string, forUpdate bool) (documentSnapshot, error) {
	query := `
SELECT id, title, source_uri, tags, metadata, content_hash, chunk_count, updated_at
FROM kb_documents
WHERE collection = $1 AND external_id = $2`
	if forUpdate {
		query += `
FOR UPDATE`
	}

	var (
		snapshot      documentSnapshot
		metadataBytes []byte
	)
	err := rower.QueryRow(ctx, query, collection, documentID).Scan(
		&snapshot.internalID,
		&snapshot.title,
		&snapshot.sourceURI,
		&snapshot.tags,
		&metadataBytes,
		&snapshot.contentHash,
		&snapshot.chunkCount,
		&snapshot.updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return documentSnapshot{}, nil
		}
		return documentSnapshot{}, fmt.Errorf("load document snapshot: %w", err)
	}

	metadataJSON, err := canonicalStoredMetadataJSON(metadataBytes)
	if err != nil {
		return documentSnapshot{}, fmt.Errorf("decode stored metadata: %w", err)
	}

	snapshot.exists = true
	snapshot.metadataJSON = metadataJSON
	return snapshot, nil
}

func documentSnapshotsEqual(left documentSnapshot, right documentSnapshot) bool {
	if left.exists != right.exists {
		return false
	}
	if !left.exists {
		return true
	}
	return left.internalID == right.internalID &&
		left.title == right.title &&
		left.sourceURI == right.sourceURI &&
		left.metadataJSON == right.metadataJSON &&
		left.contentHash == right.contentHash &&
		left.chunkCount == right.chunkCount &&
		left.updatedAt.Equal(right.updatedAt) &&
		slices.Equal(left.tags, right.tags)
}

func canonicalMetadataJSON(metadata map[string]any) (string, error) {
	bytes, err := json.Marshal(normalizeMetadata(metadata))
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func canonicalStoredMetadataJSON(raw []byte) (string, error) {
	if len(raw) == 0 {
		return canonicalMetadataJSON(nil)
	}

	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return "", err
	}
	return canonicalMetadataJSON(metadata)
}

func documentLockKey(collection string, documentID string) int64 {
	sum := sha256.Sum256([]byte(collection + "\x00" + documentID))
	return int64(binary.BigEndian.Uint64(sum[:8]))
}

func newDocumentStats(req UpsertDocumentRequest, contentHash string, chunkCount int) DocumentStats {
	return DocumentStats{
		Collection:  req.Collection,
		DocumentID:  req.DocumentID,
		ContentHash: contentHash,
		ChunkCount:  chunkCount,
	}
}

func (s *Store) normalizeDocumentRequest(req UpsertDocumentRequest) UpsertDocumentRequest {
	req.Collection = s.resolveCollection(req.Collection)
	req.DocumentID = strings.TrimSpace(req.DocumentID)
	req.Title = strings.TrimSpace(req.Title)
	req.Content = normalizeText(req.Content)
	req.SourceURI = strings.TrimSpace(req.SourceURI)
	req.Tags = normalizeTags(req.Tags)
	req.Metadata = normalizeMetadata(req.Metadata)
	return req
}

func (s *Store) validateDocumentRequest(req UpsertDocumentRequest) error {
	if req.DocumentID == "" {
		return errors.New("document id is required")
	}
	if req.Content == "" {
		return errors.New("content is required")
	}
	return nil
}

func (s *Store) normalizeSearchRequest(req SearchRequest) SearchRequest {
	req.Collection = s.resolveCollection(req.Collection)
	req.Query = strings.TrimSpace(req.Query)
	if req.Limit <= 0 {
		req.Limit = s.cfg.DefaultSearchLimit
	}
	if req.CandidateLimit <= 0 {
		req.CandidateLimit = s.cfg.CandidateLimit
	}
	if req.Mode == "" {
		req.Mode = SearchModeHybrid
	}
	req.MetadataFilter = normalizeMetadata(req.MetadataFilter)
	return req
}

func (s *Store) validateSearchRequest(req SearchRequest) error {
	if req.Query == "" {
		return errors.New("query is required")
	}
	switch req.Mode {
	case SearchModeHybrid, SearchModeKeyword, SearchModeVector:
	default:
		return fmt.Errorf("unsupported search mode %q", req.Mode)
	}
	if req.Limit <= 0 {
		return errors.New("limit must be greater than 0")
	}
	if req.CandidateLimit < req.Limit {
		return errors.New("candidate limit must be greater than or equal to limit")
	}
	if _, err := encodeMetadataFilter(req.MetadataFilter); err != nil {
		return fmt.Errorf("invalid metadata filter: %w", err)
	}
	return nil
}

func (s *Store) resolveCollection(collection string) string {
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return s.cfg.DefaultCollection
	}
	return collection
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func normalizeMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func encodeMetadataFilter(metadata map[string]any) (string, error) {
	bytes, err := json.Marshal(normalizeMetadata(metadata))
	if err != nil {
		return "", fmt.Errorf("marshal metadata filter: %w", err)
	}
	return string(bytes), nil
}

func (s *Store) ensureReady() error {
	if s == nil || s.pool == nil {
		return errors.New("store is not initialized")
	}
	return nil
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func chunkKey(documentID string, ordinal int) string {
	return fmt.Sprintf("%s:%06d", documentID, ordinal)
}

func vectorLiteral(vector []float32) string {
	parts := make([]string, len(vector))
	for i, value := range vector {
		parts[i] = strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.8f", value), "0"), ".")
		if parts[i] == "" {
			parts[i] = "0"
		}
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func ensureEmbeddingDimensions(ctx context.Context, tx pgx.Tx, expected int) error {
	var typeName string
	err := tx.QueryRow(ctx, `
SELECT pg_catalog.format_type(a.atttypid, a.atttypmod)
FROM pg_catalog.pg_attribute a
JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = current_schema()
  AND c.relname = 'kb_chunks'
  AND a.attname = 'embedding'
  AND a.attnum > 0
  AND NOT a.attisdropped
`).Scan(&typeName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("kb_chunks.embedding column is missing")
		}
		return fmt.Errorf("load embedding column type: %w", err)
	}

	actual, err := parseVectorDimensions(typeName)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("schema embedding dimension mismatch: database=%d config=%d", actual, expected)
	}
	return nil
}

func parseVectorDimensions(typeName string) (int, error) {
	const prefix = "vector("
	if !strings.HasPrefix(typeName, prefix) || !strings.HasSuffix(typeName, ")") {
		return 0, fmt.Errorf("unexpected embedding column type %q", typeName)
	}

	raw := strings.TrimSuffix(strings.TrimPrefix(typeName, prefix), ")")
	dimensions, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse embedding dimensions from %q: %w", typeName, err)
	}
	if dimensions <= 0 {
		return 0, fmt.Errorf("embedding dimensions must be greater than 0, got %d", dimensions)
	}
	return dimensions, nil
}

func ensureRequiredExtensions(ctx context.Context, tx pgx.Tx) error {
	requiredExtensions := sdkmeta.RequiredExtensionNames()
	rows, err := tx.Query(ctx, `
SELECT name
FROM pg_catalog.pg_available_extensions
WHERE name = ANY($1)
`, requiredExtensions)
	if err != nil {
		return fmt.Errorf("load required extension support: %w", err)
	}
	defer rows.Close()

	available := make(map[string]struct{}, 2)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scan required extension support: %w", err)
		}
		available[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate required extension support: %w", err)
	}
	return validateRequiredExtensions(available)
}

func validateRequiredExtensions(available map[string]struct{}) error {
	var missing []string
	for _, name := range sdkmeta.RequiredExtensionNames() {
		if _, ok := available[name]; ok {
			continue
		}
		missing = append(missing, name)
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf(
		"database is missing required extension support for %s; simplykb expects ParadeDB with pgvector support, not plain Postgres",
		strings.Join(missing, ", "),
	)
}
