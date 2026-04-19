package exampleenv

import (
	"net/url"
	"strings"
	"testing"
)

func TestDefaultDatabaseURLPrefersExplicitURL(t *testing.T) {
	t.Setenv("SIMPLYKB_DATABASE_URL", "postgres://custom/custom")
	t.Setenv("PARADEDB_PORT", "35432")
	t.Setenv("POSTGRES_USER", "other")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_DB", "demo")

	if got := DefaultDatabaseURL(); got != "postgres://custom/custom" {
		t.Fatalf("DefaultDatabaseURL() = %q, want explicit env url", got)
	}
}

func TestDefaultDatabaseURLEscapesCredentialsAndDatabaseName(t *testing.T) {
	t.Setenv("SIMPLYKB_DATABASE_URL", "")
	t.Setenv("PARADEDB_PORT", "35432")
	t.Setenv("POSTGRES_USER", "demo user")
	t.Setenv("POSTGRES_PASSWORD", "sec:ret@1")
	t.Setenv("POSTGRES_DB", "knowledge/base?")

	got := DefaultDatabaseURL()
	if strings.Contains(got, "demo user") {
		t.Fatalf("expected username to be escaped in %q", got)
	}
	if strings.Contains(got, "sec:ret@1") {
		t.Fatalf("expected password to be escaped in %q", got)
	}
	if strings.Contains(got, "/knowledge/base") {
		t.Fatalf("expected database name to be escaped in %q", got)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Scheme != "postgres" {
		t.Fatalf("Scheme = %q", parsed.Scheme)
	}
	if parsed.Host != "localhost:35432" {
		t.Fatalf("Host = %q", parsed.Host)
	}
	if gotUser := parsed.User.Username(); gotUser != "demo user" {
		t.Fatalf("Username = %q", gotUser)
	}
	gotPassword, ok := parsed.User.Password()
	if !ok || gotPassword != "sec:ret@1" {
		t.Fatalf("Password = %q, ok = %v", gotPassword, ok)
	}
	if gotDatabase := strings.TrimPrefix(parsed.Path, "/"); gotDatabase != "knowledge/base?" {
		t.Fatalf("Database = %q", gotDatabase)
	}
	if parsed.Query().Get("sslmode") != "disable" {
		t.Fatalf("sslmode = %q", parsed.Query().Get("sslmode"))
	}
}
