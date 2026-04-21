package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func DatabaseURL(tb testing.TB) string {
	tb.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("SIMPLYKB_DATABASE_URL"))
	if databaseURL == "" {
		tb.Skip("SIMPLYKB_DATABASE_URL is not set")
	}
	return databaseURL
}

func CreateSchema(tb testing.TB, databaseURL string, prefix string) string {
	tb.Helper()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		tb.Fatalf("connect admin pool: %v", err)
	}
	defer pool.Close()

	schema := fmt.Sprintf("%s_%d", sanitizeIdentifier(prefix), time.Now().UnixNano())
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		tb.Fatalf("create schema %s: %v", schema, err)
	}

	tb.Cleanup(func() {
		cleanupPool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			tb.Fatalf("connect cleanup pool: %v", err)
		}
		defer cleanupPool.Close()
		if _, err := cleanupPool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			tb.Fatalf("drop schema %s: %v", schema, err)
		}
	})

	return schema
}

func URLWithSearchPath(tb testing.TB, databaseURL string, schema string) string {
	tb.Helper()

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		tb.Fatalf("parse database url: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schema+",public")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func sanitizeIdentifier(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return "simplykb_test"
	}

	var builder strings.Builder
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}

	sanitized := strings.Trim(builder.String(), "_")
	if sanitized == "" {
		return "simplykb_test"
	}
	return sanitized
}
