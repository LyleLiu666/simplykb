package main

import "testing"

func TestDefaultDatabaseURLPrefersExplicitURL(t *testing.T) {
	t.Setenv("SIMPLYKB_DATABASE_URL", "postgres://custom/custom")
	t.Setenv("PARADEDB_PORT", "35432")
	t.Setenv("POSTGRES_USER", "other")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_DB", "demo")

	if got := defaultDatabaseURL(); got != "postgres://custom/custom" {
		t.Fatalf("defaultDatabaseURL() = %q, want explicit env url", got)
	}
}

func TestDefaultDatabaseURLBuildsFromLocalEnv(t *testing.T) {
	t.Setenv("SIMPLYKB_DATABASE_URL", "")
	t.Setenv("PARADEDB_PORT", "35432")
	t.Setenv("POSTGRES_USER", "demo")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_DB", "knowledge")

	want := "postgres://demo:secret@localhost:35432/knowledge?sslmode=disable"
	if got := defaultDatabaseURL(); got != want {
		t.Fatalf("defaultDatabaseURL() = %q, want %q", got, want)
	}
}

func TestDefaultDatabaseURLFallsBackToProjectDefaults(t *testing.T) {
	t.Setenv("SIMPLYKB_DATABASE_URL", "")
	t.Setenv("PARADEDB_PORT", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("POSTGRES_DB", "")

	want := "postgres://simplykb:simplykb@localhost:25432/simplykb?sslmode=disable"
	if got := defaultDatabaseURL(); got != want {
		t.Fatalf("defaultDatabaseURL() = %q, want %q", got, want)
	}
}
