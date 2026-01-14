package annotation

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

func GetDatabase(filename string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrency (allows reads during writes)
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		if err := db.Close(); err != nil {
			return nil, err
		}
		return nil, err
	}

	// Set busy timeout to 5 seconds (wait instead of immediately failing with SQLITE_BUSY)
	_, err = db.Exec("PRAGMA busy_timeout=5000")
	if err != nil {
		if err := db.Close(); err != nil {
			return nil, err
		}
		return nil, err
	}

	// With WAL mode + busy_timeout, SQLite handles concurrency correctly
	// No need to artificially limit connections - let database/sql use defaults

	return db, nil
}
