package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/LyleLiu666/simplykb/examples/internal/exampleenv"
)

func main() {
	ctx := context.Background()

	report, err := exampleenv.CollectDatabaseDiagnostics(ctx, exampleenv.DefaultDatabaseURL())
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect database diagnostics: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("status: %s\n", report.Status())
	fmt.Printf("database url: %s\n", report.RedactedDatabaseURL())
	fmt.Printf("current database: %s\n", report.CurrentDatabase)
	fmt.Printf("current schema: %s\n", report.CurrentSchema)
	fmt.Printf("search path: %s\n", report.SearchPath)
	fmt.Printf("current user: %s\n", report.CurrentUser)
	fmt.Printf("server version: %s\n", report.ServerVersion)
	fmt.Println("required extensions:")
	for _, name := range []string{"pg_search", "vector"} {
		state := "missing"
		if report.RequiredExtensions[name] {
			state = "available"
		}
		fmt.Printf("- %s: %s\n", name, state)
	}
	fmt.Printf("migrations table: %s\n", presentState(report.MigrationsTableExists))
	if report.MigrationsTableExists {
		fmt.Printf("applied migrations: %s\n", joinVersions(report.AppliedMigrationVersions))
		fmt.Printf("latest migration: %d\n", report.LatestMigrationVersion)
		fmt.Printf("expected latest migration: %d\n", report.ExpectedMigrationVersion)
	}
	if report.EmbeddingColumnType != "" {
		fmt.Printf("embedding column type: %s\n", report.EmbeddingColumnType)
	}
	fmt.Printf("documents: %d\n", report.DocumentCount)
	fmt.Printf("chunks: %d\n", report.ChunkCount)

	if len(report.MissingExtensions) > 0 {
		fmt.Fprintf(os.Stderr, "database is missing required extension support for %s\n", strings.Join(report.MissingExtensions, ", "))
		os.Exit(1)
	}
}

func presentState(ok bool) string {
	if ok {
		return "present"
	}
	return "absent"
}

func joinVersions(versions []int64) string {
	if len(versions) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(versions))
	for _, version := range versions {
		parts = append(parts, fmt.Sprintf("%d", version))
	}
	return strings.Join(parts, ", ")
}
