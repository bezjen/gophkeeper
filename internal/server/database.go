package server

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func InitDatabase(driver, dsn string) (*sql.DB, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	if err := runMigrations(driver, dsn); err != nil {
		return nil, err
	}

	log.Printf("Connected to database using %s driver", driver)
	return db, nil
}

func runMigrations(driver, dsn string) error {
	var databaseURL string

	switch driver {
	case "sqlite":
		databaseURL = fmt.Sprintf("sqlite://%s", dsn)
	case "postgres":
		databaseURL = dsn
	default:
		return fmt.Errorf("unsupported database driver: %s", driver)
	}

	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
