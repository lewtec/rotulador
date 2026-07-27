package annotation

import (
	"context"
	"database/sql"
	"fmt"

	appdb "github.com/lewtec/rotulador/db"
	_ "modernc.org/sqlite"
)

// GetDatabase opens a SQLite database with project-standard PRAGMAs applied
// on every connection.
func GetDatabase(filename string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", appdb.SQLiteOpenDSN(filename))
	if err != nil {
		return nil, err
	}

	// Force a real connection so PRAGMAs apply and bad paths fail early.
	if err := conn.Ping(); err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			ReportError(context.Background(), closeErr, "msg", "failed to close database after ping failure")
		}
		return nil, fmt.Errorf("sqlite open/ping: %w", err)
	}

	return conn, nil
}
