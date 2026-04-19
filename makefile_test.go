package simplykb

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMakePrintDBURLUsesEscapedDefaultDatabaseURL(t *testing.T) {
	got := makeTargetOutput(t, "print-db-url",
		"SIMPLYKB_DATABASE_URL=",
		"POSTGRES_USER=demo user",
		"POSTGRES_PASSWORD=sec:ret@1",
		"POSTGRES_DB=knowledge/base?",
		"PARADEDB_PORT=35432",
	)
	wantURL := "postgres://demo%20user:sec%3Aret%401@localhost:35432/knowledge%2Fbase%3F?sslmode=disable"
	if strings.TrimSpace(got) != wantURL {
		t.Fatalf("make print-db-url = %q, want %q", strings.TrimSpace(got), wantURL)
	}
	if strings.Contains(got, "postgres://demo user:sec:ret@1@localhost") {
		t.Fatalf("make print-db-url still uses a raw interpolated URL\n%s", got)
	}
}

func TestMakePrintDBURLPreservesDollarSignsInGeneratedURL(t *testing.T) {
	got := makeTargetOutput(t, "print-db-url",
		"SIMPLYKB_DATABASE_URL=",
		"POSTGRES_USER=demo$user",
		"POSTGRES_PASSWORD=sec$ret",
		"POSTGRES_DB=knowledge$db",
		"PARADEDB_PORT=35432",
	)
	wantURL := "postgres://demo$user:sec$ret@localhost:35432/knowledge$db?sslmode=disable"
	if strings.TrimSpace(got) != wantURL {
		t.Fatalf("make print-db-url = %q, want %q", strings.TrimSpace(got), wantURL)
	}
}

func TestMakePrintDBURLPreservesDollarSignsInExplicitDBURL(t *testing.T) {
	wantURL := "postgres://u:p@localhost/db?application_name=$USER"
	got := makeTargetOutput(t, "print-db-url",
		"DB_URL="+wantURL,
	)
	if strings.TrimSpace(got) != wantURL {
		t.Fatalf("make print-db-url = %q, want %q", strings.TrimSpace(got), wantURL)
	}
}

func makeTargetOutput(t *testing.T, target string, env ...string) string {
	t.Helper()

	cmd := exec.Command("make", target)
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make %s error = %v\n%s", target, err, output)
	}
	return string(output)
}
