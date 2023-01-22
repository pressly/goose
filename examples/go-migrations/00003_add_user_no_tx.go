package main

import (
	"database/sql"
	"errors"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(Up00003, Down00003)
}

func Up00003(db *sql.DB) error {
	id, err := getUserID(db, "jamesbond")
	if err != nil {
		return err
	}
	if id == 0 {
		query := "INSERT INTO users (username, name, surname) VALUES ($1, $2, $3)"
		if _, err := db.Exec(query, "jamesbond", "James", "Bond"); err != nil {
			return err
		}
	}
	return nil
}

func getUserID(db *sql.DB, username string) (int, error) {
	var id int
	err := db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	return id, nil
}

func Down00003(db *sql.DB) error {
	query := "DELETE FROM users WHERE username = $1"
	if _, err := db.Exec(query, "jamesbond"); err != nil {
		return err
	}
	return nil
}
