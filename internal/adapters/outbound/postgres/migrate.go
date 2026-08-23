package postgres

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	// Blank imports register golang-migrate's postgres database driver and
	// file source driver.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies every pending migration in migrationsPath (a
// "file://" source directory) against databaseURL.
func RunMigrations(databaseURL, migrationsPath string) error {
	m, err := migrate.New(fmt.Sprintf("file://%s", migrationsPath), databaseURL)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
