package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(ctx context.Context, connString, dbName string, migrationsSourceURL string) error {
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return fmt.Errorf("could not open database connection: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("could not establish connection with databse: %w", err)
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationsSourceURL,
		dbName,
		driver,
	)
	if err != nil {
		return fmt.Errorf("could not create migration instance: %w", err)
	}

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("could not run migrations down: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("could not run migrations up: %w", err)
	}
	return nil
}
