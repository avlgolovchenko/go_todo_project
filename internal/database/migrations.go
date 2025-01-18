package database

import (
	"database/sql"
	"log"
)

func RunMigrations(db *sql.DB) error {
	log.Println("Выполнение миграций базы данных...")

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS scheduler (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		title TEXT NOT NULL,
		comment TEXT,
		repeat TEXT CHECK (length(repeat) <= 128)
	);
	CREATE INDEX IF NOT EXISTS idx_date ON scheduler (date);
	`
	_, err := db.Exec(createTableQuery)
	if err != nil {
		return err
	}

	log.Println("Миграции выполнены успешно.")
	return nil
}
