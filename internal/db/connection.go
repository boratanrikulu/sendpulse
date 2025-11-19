package db

import (
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	_ "github.com/uptrace/bun/driver/pgdriver" // PostgreSQL driver
)

// Connect returns a DB connection.
func Connect(dsn string) (*bun.DB, error) {
	sqldb, err := sql.Open("pg", dsn)
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := sqldb.Ping(); err != nil {
		sqldb.Close()
		return nil, err
	}

	db := bun.NewDB(sqldb, pgdialect.New())
	return db, nil
}
