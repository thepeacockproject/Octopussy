// Package store is Octopussy's persistence layer. It is built on GORM so the
// same model definitions and query code work across database backends, with
// gormigrate providing Rails-style versioned up/down migrations. SQLite (the
// pure-Go glebarez driver, so no cgo) is the default; the dialect is selected
// by Config.Driver, leaving room to add Postgres/MySQL later.
package store

import (
	"fmt"
	"strings"

	"github.com/glebarez/sqlite"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config selects the database backend. Driver picks the dialect ("sqlite" by
// default); DSN is the driver-specific connection string (for SQLite, a file
// path or ":memory:").
type Config struct {
	Driver string
	DSN    string
	// Debug enables verbose GORM query logging; otherwise only warnings/errors.
	Debug bool
}

// Store wraps the GORM handle. It is safe for concurrent use.
type Store struct {
	db *gorm.DB
}

// Open connects to the configured database, runs pending migrations, and
// returns a ready Store.
func Open(cfg Config) (*Store, error) {
	dialector, isMemory, err := dialectorFor(cfg)
	if err != nil {
		return nil, err
	}

	logLevel := logger.Warn
	if cfg.Debug {
		logLevel = logger.Info
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// A ":memory:" SQLite database lives inside a single connection; if the pool
	// opened a second connection it would see an empty, separate database. Pin
	// the pool to one connection so migrations and queries share the same DB.
	if isMemory {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxOpenConns(1)
	}

	if err := migrate(db); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

// dialectorFor maps a Config to a GORM dialector. It reports whether the target
// is an in-memory SQLite database so the caller can pin the connection pool.
func dialectorFor(cfg Config) (gorm.Dialector, bool, error) {
	switch cfg.Driver {
	case "", "sqlite":
		dsn := cfg.DSN
		if dsn == "" {
			dsn = "octopussy.db"
		}
		isMemory := strings.Contains(dsn, ":memory:") || strings.Contains(dsn, "mode=memory")
		return sqlite.Open(dsn), isMemory, nil
	default:
		// Future: add gorm.io/driver/postgres, gorm.io/driver/mysql here.
		return nil, false, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}
}

// migrate applies every pending migration in order, tracked in a migrations
// table so each runs exactly once.
func migrate(db *gorm.DB) error {
	m := gormigrate.New(db, gormigrate.DefaultOptions, migrations())
	if err := m.Migrate(); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// Close releases the underlying database connections.
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
