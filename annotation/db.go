package annotation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// GetDatabase opens a SQLite database with connection-level pragmas applied on
// every pooled connection (via modernc URI _pragma params):
//   - foreign_keys=ON so schema FKs (e.g. annotations → images) are enforced
//   - busy_timeout so concurrent writers wait instead of failing immediately
//   - journal_mode=WAL for better concurrent read/write
//
// Setting these only with db.Exec would affect a single pool connection.
func GetDatabase(filename string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", sqliteDSN(filename))
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			ReportError(context.Background(), closeErr, "msg", "failed to close database after ping failure")
		}
		return nil, err
	}

	return db, nil
}

// sqliteDSN builds a modernc.org/sqlite URI that applies pragmas on each new
// connection drawn from database/sql's pool.
func sqliteDSN(filename string) string {
	// busy_timeout + foreign_keys apply to every connection.
	// journal_mode=WAL is only for file-backed DBs (WAL is unsupported on pure :memory:).
	const core = "_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	const withWAL = core + "&_pragma=journal_mode(WAL)"

	switch {
	case filename == "" || filename == ":memory:":
		// In-memory DB: URI form required for query params; no WAL.
		return "file::memory:?" + core
	case strings.HasPrefix(filename, "file:"):
		if strings.Contains(filename, "?") {
			return filename + "&" + withWAL
		}
		return filename + "?" + withWAL
	case strings.Contains(filename, "?"):
		// Bare path that already has a query string — rare; append pragmas.
		return filename + "&" + withWAL
	default:
		// Relative or absolute filesystem path.
		return fmt.Sprintf("file:%s?%s", filename, withWAL)
	}
}
