package exampleenv

import (
	"net"
	"net/url"
	"os"
	"strings"
)

const (
	defaultLocalUser = "simplykb"
	defaultLocalPass = "simplykb"
	defaultLocalDB   = "simplykb"
	defaultLocalPort = "25432"
)

func DefaultDatabaseURL() string {
	if databaseURL := strings.TrimSpace(os.Getenv("SIMPLYKB_DATABASE_URL")); databaseURL != "" {
		return databaseURL
	}

	user := StringOrDefault("POSTGRES_USER", defaultLocalUser)
	password := StringOrDefault("POSTGRES_PASSWORD", defaultLocalPass)
	database := StringOrDefault("POSTGRES_DB", defaultLocalDB)
	port := StringOrDefault("PARADEDB_PORT", defaultLocalPort)

	connectionURL := &url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort("localhost", port),
		RawQuery: "sslmode=disable",
		User:     url.UserPassword(user, password),
	}
	connectionURL.Path = "/" + database
	connectionURL.RawPath = "/" + url.PathEscape(database)
	return connectionURL.String()
}

func StringOrDefault(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
