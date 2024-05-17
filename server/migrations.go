package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetBootstrapSQLMigrations() []string {
	return []string{
		`CREATE TABLE MigrationHead (
			migration_id INT NOT NULL DEFAULT -1
		)`,
		`INSERT INTO MigrationHead (migration_id) VALUES (-1)`,
	}
}

// TODO: can we update this to read from schema?
func GetSQLMigrations() []string {
	return []string{
		`CREATE TABLE Users (
			email VARCHAR(256) NOT NULL
			PRIMARY KEY (email)
		)`,
	}
}

func RunMigrations(ctx context.Context, logger *slog.Logger, db *pgxpool.Pool) error {
	// the first migration is the migration head. We need to see which migration has been applied and start from there.
	var migration_id int
	row := db.QueryRow(ctx, "SELECT migration_id FROM MigrationHead")
	err := row.Scan(&migration_id)
	if err != nil {

		logger.Info("MigrationHead doesn't exist, bootstrapping the db with the MigrationHead table")
		bootstrap_sql := GetBootstrapSQLMigrations()

		for _, migration := range bootstrap_sql {
			_, err := db.Exec(ctx, migration)
			if err != nil {
				return fmt.Errorf("failed to bootstrap db: %w", err)
			}
		}
		migration_id = -1
	}
	migrations := GetSQLMigrations()
	for i, migration := range migrations {
		if i <= migration_id {
			continue
		}
		logger.Info(fmt.Sprintf("applying migration %d: %s\n", i, migration))
		_, err := db.Exec(ctx, migration)
		if err != nil {
			return fmt.Errorf("failed at migration %d: %w", i, err)
		}

		_, err = db.Exec(ctx, "UPDATE MigrationHead SET migration_id = ?", i)
		if err != nil {
			return fmt.Errorf("failed to update migration head for migration %d: %w", i, err)
		}
	}
	return nil
}
