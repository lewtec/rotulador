package annotation

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "modernc.org/sqlite"
)

// sqliteOpenDSN builds a modernc.org/sqlite data source name that applies
// connection-scoped PRAGMAs on every new connection from the pool.
//
// Without DSN-level PRAGMAs, a one-shot db.Exec only configures the single
// connection that ran it; later pooled connections would miss foreign_keys
// and busy_timeout (SQLite defaults foreign_keys to OFF).
func sqliteOpenDSN(filename string) string {
	q := url.Values{}
	// Enforce FK constraints declared in schema (including ON DELETE CASCADE).
	q.Add("_pragma", "foreign_keys(1)")
	// Wait up to 5s on lock contention instead of failing with SQLITE_BUSY.
	q.Add("_pragma", "busy_timeout(5000)")
	// WAL improves concurrent readers during writes; safe to set per connection.
	q.Add("_pragma", "journal_mode(WAL)")
	return filename + "?" + q.Encode()
}

// GetDatabase opens a SQLite database with project-standard PRAGMAs applied
// on every connection.
func GetDatabase(filename string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", sqliteOpenDSN(filename))
	if err != nil {
		return nil, err
	}

	// Force a real connection so PRAGMAs apply and bad paths fail early.
	if err := db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			ReportError(context.Background(), closeErr, "msg", "failed to close database after ping failure")
		}
		return nil, fmt.Errorf("sqlite open/ping: %w", err)
	}

	return db, nil
}
