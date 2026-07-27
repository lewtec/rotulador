package db

import "net/url"

// SQLiteOpenDSN builds a modernc.org/sqlite data source name that applies
// connection-scoped PRAGMAs on every new connection from the pool.
//
// Without DSN-level PRAGMAs, a one-shot db.Exec only configures the single
// connection that ran it; later pooled connections would miss foreign_keys
// and busy_timeout (SQLite defaults foreign_keys to OFF).
//
// journal_mode(WAL) is omitted for ":memory:" databases (unnecessary overhead).
func SQLiteOpenDSN(filename string) string {
	q := url.Values{}
	// Enforce FK constraints declared in schema (including ON DELETE CASCADE).
	q.Add("_pragma", "foreign_keys(1)")
	// Wait up to 5s on lock contention instead of failing with SQLITE_BUSY.
	q.Add("_pragma", "busy_timeout(5000)")
	if filename != ":memory:" {
		// WAL improves concurrent readers during writes; safe to set per connection.
		q.Add("_pragma", "journal_mode(WAL)")
	}
	return filename + "?" + q.Encode()
}
