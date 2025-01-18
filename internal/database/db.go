package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func ConnectDB(dbPath string) (*sql.DB, error) {
	log.Printf("Подключение к базе данных: %s", dbPath)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	return db, nil
}
