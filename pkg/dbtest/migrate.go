package dbtest

import (
	"fmt"
	"io"
	"os"

	"github.com/jmoiron/sqlx"
)

// MigrateFromFile executes all SQL queries from the files over a database
// connection.
func MigrateFromFile(db *sqlx.DB, fileNames ...string) error {
	for _, fileName := range fileNames {
		fh, err := os.Open(fileName)
		if err != nil {
			return fmt.Errorf("os.Open: %w", err)
		}

		fileBytes, err := io.ReadAll(fh)
		if err != nil {
			return fmt.Errorf("io.ReadAll: %w", err)
		}

		if err = fh.Close(); err != nil {
			return fmt.Errorf("fh.Close: %w", err)
		}

		if _, err = db.Exec(string(fileBytes)); err != nil {
			return fmt.Errorf("db.Exec: %w", err)
		}
	}

	return nil
}
