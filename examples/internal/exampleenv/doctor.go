package exampleenv

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/LyleLiu666/simplykb/internal/sdkmeta"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DatabaseDiagnostics struct {
	DatabaseURL              string
	CurrentDatabase          string
	CurrentSchema            string
	SearchPath               string
	CurrentUser              string
	ServerVersion            string
	RequiredExtensions       map[string]bool
	MissingExtensions        []string
	MigrationsTableExists    bool
	AppliedMigrationVersions []int64
	LatestMigrationVersion   int64
	ExpectedMigrationVersion int64
	EmbeddingColumnType      string
	DocumentCount            int64
	ChunkCount               int64
}

func CollectDatabaseDiagnostics(ctx context.Context, databaseURL string) (DatabaseDiagnostics, error) {
	if strings.TrimSpace(databaseURL) == "" {
		databaseURL = DefaultDatabaseURL()
	}

	report := DatabaseDiagnostics{
		DatabaseURL:              databaseURL,
		RequiredExtensions:       map[string]bool{},
		ExpectedMigrationVersion: sdkmeta.LatestSchemaMigrationVersion,
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return DatabaseDiagnostics{}, fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return DatabaseDiagnostics{}, fmt.Errorf("ping database: %w", err)
	}

	if err := pool.QueryRow(ctx, `
SELECT
	current_database(),
	current_schema(),
	current_setting('search_path'),
	current_user,
	current_setting('server_version')
`).Scan(
		&report.CurrentDatabase,
		&report.CurrentSchema,
		&report.SearchPath,
		&report.CurrentUser,
		&report.ServerVersion,
	); err != nil {
		return DatabaseDiagnostics{}, fmt.Errorf("load database identity: %w", err)
	}

	if err := loadRequiredExtensions(ctx, pool, &report); err != nil {
		return DatabaseDiagnostics{}, err
	}
	if err := loadMigrationState(ctx, pool, &report); err != nil {
		return DatabaseDiagnostics{}, err
	}
	if err := loadTableCounts(ctx, pool, &report); err != nil {
		return DatabaseDiagnostics{}, err
	}

	return report, nil
}

func (r DatabaseDiagnostics) Status() string {
	switch {
	case len(r.MissingExtensions) > 0:
		return "failed"
	case !r.MigrationsTableExists:
		return "database reachable; schema not migrated yet"
	case r.LatestMigrationVersion < r.ExpectedMigrationVersion:
		return fmt.Sprintf(
			"database reachable; schema migration is behind current version (%d < %d)",
			r.LatestMigrationVersion,
			r.ExpectedMigrationVersion,
		)
	default:
		return "ready"
	}
}

func (r DatabaseDiagnostics) RedactedDatabaseURL() string {
	parsed, err := url.Parse(r.DatabaseURL)
	if err != nil {
		return r.DatabaseURL
	}
	if parsed.User == nil {
		return parsed.String()
	}
	username := parsed.User.Username()
	if _, hasPassword := parsed.User.Password(); hasPassword {
		parsed.User = url.UserPassword(username, "xxxxx")
		return parsed.String()
	}
	parsed.User = url.User(username)
	return parsed.String()
}

func loadRequiredExtensions(ctx context.Context, pool *pgxpool.Pool, report *DatabaseDiagnostics) error {
	requiredExtensions := sdkmeta.RequiredExtensionNames()
	rows, err := pool.Query(ctx, `
SELECT name
FROM pg_catalog.pg_available_extensions
WHERE name = ANY($1)
`, requiredExtensions)
	if err != nil {
		return fmt.Errorf("load required extension support: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scan required extension support: %w", err)
		}
		report.RequiredExtensions[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate required extension support: %w", err)
	}

	report.MissingExtensions = missingRequiredExtensions(report.RequiredExtensions)
	return nil
}

func loadMigrationState(ctx context.Context, pool *pgxpool.Pool, report *DatabaseDiagnostics) error {
	if err := pool.QueryRow(ctx, `
SELECT to_regclass(current_schema() || '.kb_schema_migrations') IS NOT NULL
`).Scan(&report.MigrationsTableExists); err != nil {
		return fmt.Errorf("check migrations table: %w", err)
	}
	if !report.MigrationsTableExists {
		return nil
	}

	rows, err := pool.Query(ctx, `
SELECT version
FROM kb_schema_migrations
ORDER BY version
`)
	if err != nil {
		return fmt.Errorf("load applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("scan applied migration: %w", err)
		}
		report.AppliedMigrationVersions = append(report.AppliedMigrationVersions, version)
		report.LatestMigrationVersion = version
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate applied migrations: %w", err)
	}

	var chunksTableExists bool
	if err := pool.QueryRow(ctx, `
SELECT to_regclass(current_schema() || '.kb_chunks') IS NOT NULL
`).Scan(&chunksTableExists); err != nil {
		return fmt.Errorf("check chunks table: %w", err)
	}
	if !chunksTableExists {
		return nil
	}

	err = pool.QueryRow(ctx, `
SELECT pg_catalog.format_type(a.atttypid, a.atttypmod)
FROM pg_catalog.pg_attribute a
JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = current_schema()
  AND c.relname = 'kb_chunks'
  AND a.attname = 'embedding'
  AND a.attnum > 0
  AND NOT a.attisdropped
`).Scan(&report.EmbeddingColumnType)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("load embedding column type: %w", err)
	}

	return nil
}

func loadTableCounts(ctx context.Context, pool *pgxpool.Pool, report *DatabaseDiagnostics) error {
	var documentsTableExists bool
	if err := pool.QueryRow(ctx, `
SELECT to_regclass(current_schema() || '.kb_documents') IS NOT NULL
`).Scan(&documentsTableExists); err != nil {
		return fmt.Errorf("check documents table: %w", err)
	}
	if documentsTableExists {
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM kb_documents`).Scan(&report.DocumentCount); err != nil {
			return fmt.Errorf("count documents: %w", err)
		}
	}

	var chunksTableExists bool
	if err := pool.QueryRow(ctx, `
SELECT to_regclass(current_schema() || '.kb_chunks') IS NOT NULL
`).Scan(&chunksTableExists); err != nil {
		return fmt.Errorf("check chunks table for count: %w", err)
	}
	if chunksTableExists {
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM kb_chunks`).Scan(&report.ChunkCount); err != nil {
			return fmt.Errorf("count chunks: %w", err)
		}
	}

	return nil
}

func missingRequiredExtensions(available map[string]bool) []string {
	requiredExtensions := sdkmeta.RequiredExtensionNames()
	missing := make([]string, 0, len(requiredExtensions))
	for _, name := range requiredExtensions {
		if available[name] {
			continue
		}
		missing = append(missing, name)
	}
	slices.Sort(missing)
	return missing
}
