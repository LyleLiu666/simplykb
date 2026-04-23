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

func TestMakeDoctorDoesNotStartDatabase(t *testing.T) {
	got := makeCommandOutput(t, []string{"-n", "doctor"})
	if strings.Contains(got, "docker compose up -d") {
		t.Fatalf("make doctor should not bootstrap Docker in dry-run output\n%s", got)
	}
	if !strings.Contains(got, "go run ./examples/internal/exampleenv/cmd/doctor") {
		t.Fatalf("make doctor dry-run should invoke the doctor command\n%s", got)
	}
}

func TestMakeDBUpDryRunChecksPortBeforeStartingDocker(t *testing.T) {
	got := makeCommandOutput(t, []string{"-n", "db-up"})
	portCheck := "lsof -nP -iTCP:"
	upCommand := "docker compose up -d"
	portCheckIndex := strings.Index(got, portCheck)
	upCommandIndex := strings.Index(got, upCommand)
	if portCheckIndex == -1 {
		t.Fatalf("make db-up dry-run should include a port preflight check\n%s", got)
	}
	if upCommandIndex == -1 {
		t.Fatalf("make db-up dry-run should still invoke docker compose up\n%s", got)
	}
	if portCheckIndex > upCommandIndex {
		t.Fatalf("make db-up dry-run should check the port before docker compose up\n%s", got)
	}
}

func makeTargetOutput(t *testing.T, target string, env ...string) string {
	t.Helper()

	return makeCommandOutput(t, []string{target}, env...)
}

func makeCommandOutput(t *testing.T, args []string, env ...string) string {
	t.Helper()

	cmd := exec.Command("make", args...)
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make %s error = %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
