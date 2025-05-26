package db

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var DB *sqlx.DB

func Init() {
	var err error
	connStr := "postgres://user:password@db:5432/testdb?sslmode=disable"
	DB, err = sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}

	schema := `
    CREATE TABLE IF NOT EXISTS primary_games (
        namespace TEXT NOT NULL,
        date INTEGER NOT NULL,
        team_a TEXT NOT NULL,
        team_b TEXT NOT NULL
    );

    CREATE INDEX IF NOT EXISTS idx_namespace_date
    ON primary_games(namespace, date);

    CREATE TABLE IF NOT EXISTS mapping (
        namespace TEXT NOT NULL,
        secondary TEXT NOT NULL,
        primry TEXT NOT NULL,
        PRIMARY KEY (namespace, secondary)
    );
    `
	DB.MustExec(schema)
}
