package database

import (
	"embed"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations runs all pending database migrations using the provided embedded filesystem.
func RunMigrations(databaseURL string, migrationsFS embed.FS) error {
	source, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, "pgx5://"+stripScheme(databaseURL))
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	version, dirty, _ := m.Version()
	if err == migrate.ErrNoChange {
		slog.Info("Database migrations up to date", "version", version)
	} else {
		slog.Info("Database migrations applied", "version", version, "dirty", dirty)
	}

	return nil
}

// stripScheme removes postgres:// or postgresql:// prefix since the pgx5 driver uses its own scheme.
func stripScheme(url string) string {
	for _, prefix := range []string{"postgresql://", "postgres://"} {
		if len(url) > len(prefix) && url[:len(prefix)] == prefix {
			return url[len(prefix):]
		}
	}
	return url
}
