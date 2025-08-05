package storage

import (
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/lib/pq"
)

//go:embed init.sql
var sqlFiles embed.FS

func InitDB(db *sql.DB) error {
	sqlBytes, err := sqlFiles.ReadFile("init.sql")
	if err != nil {
		return fmt.Errorf("failed to read init.sql: %w", err)
	}

	_, err = db.Exec(string(sqlBytes))
	if err != nil {
		return fmt.Errorf("failed to execute init.sql: %w", err)
	}

	fmt.Println("Database initialized successfully")
	return nil
}
