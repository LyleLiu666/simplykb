package simplykb

import (
	"fmt"

	"github.com/LyleLiu666/simplykb/internal/sdkmeta"
)

type migration struct {
	version int64
	name    string
	sql     string
}

func bootstrapMigrationSQL() string {
	return `
CREATE TABLE IF NOT EXISTS kb_schema_migrations (
    version BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`
}

func schemaMigrations(dimensions int) []migration {
	return []migration{
		{
			version: sdkmeta.MigrationVersionBaseTables,
			name:    "base_tables",
			sql: fmt.Sprintf(`
CREATE EXTENSION IF NOT EXISTS pg_search;
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS kb_documents (
    id BIGSERIAL PRIMARY KEY,
    collection TEXT NOT NULL,
    external_id TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    source_uri TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_hash TEXT NOT NULL,
    chunk_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (collection, external_id)
);

CREATE TABLE IF NOT EXISTS kb_chunks (
    id BIGSERIAL PRIMARY KEY,
    collection TEXT NOT NULL,
    document_id BIGINT NOT NULL REFERENCES kb_documents(id) ON DELETE CASCADE,
    document_external_id TEXT NOT NULL,
    chunk_key TEXT NOT NULL,
    chunk_no INTEGER NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    search_text TEXT GENERATED ALWAYS AS (btrim(coalesce(title, '') || ' ' || coalesce(content, ''))) STORED,
    source_uri TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    embedding vector(%d) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (collection, document_external_id, chunk_no),
    UNIQUE (collection, chunk_key)
);`, dimensions),
		},
		{
			version: sdkmeta.MigrationVersionIndexes,
			name:    "indexes",
			sql: `
CREATE INDEX IF NOT EXISTS kb_chunks_embedding_idx
    ON kb_chunks USING hnsw (embedding vector_cosine_ops);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_indexes
        WHERE schemaname = current_schema()
          AND indexname = 'kb_chunks_bm25_idx'
    ) THEN
        EXECUTE '
            CREATE INDEX kb_chunks_bm25_idx
            ON kb_chunks
            USING bm25 (id, collection, document_external_id, chunk_key, title, search_text)
            WITH (key_field=''id'')
        ';
    END IF;
END $$;`,
		},
		{
			version: sdkmeta.MigrationVersionDropRedundantIndex,
			name:    "drop_redundant_indexes",
			sql: `
DROP INDEX IF EXISTS kb_documents_collection_idx;
DROP INDEX IF EXISTS kb_chunks_collection_idx;
DROP INDEX IF EXISTS kb_chunks_key_idx;`,
		},
		{
			version: sdkmeta.MigrationVersionMetadataFilterIndex,
			name:    "metadata_filter_index",
			sql: `
CREATE INDEX IF NOT EXISTS kb_chunks_metadata_idx
    ON kb_chunks USING gin (metadata jsonb_path_ops);`,
		},
	}
}
