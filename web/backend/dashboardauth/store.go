//go:build !mipsle && !netbsd && !(freebsd && arm)

// Package dashboardauth provides a bcrypt-backed SQLite store for the
// launcher dashboard password. The database contains a single row (id=1)
// with the bcrypt hash; no plaintext is ever persisted.
package dashboardauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite" // register "sqlite" driver
)

// Store holds a handle to the SQLite database that stores the bcrypt hash.
type Store struct {
	db   *sql.DB
	path string // absolute path to the SQLite file
}

// New opens (or creates) the database inside dir, using the package's
// canonical filename. This is the preferred constructor for most callers.
// Any error is wrapped with the resolved path so callers get actionable output.
func New(dir string) (*Store, error) {
	path := filepath.Join(dir, DBFilename)
	s, err := Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	return s, nil
}

// Open opens (or creates) the SQLite database at path and migrates the schema.
func Open(path string) (*Store, error) {
	db, err := sql.Open(sqliteDriver, path)
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec(sqlCreateTable); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db, path: path}, nil
}

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

// DBPath returns the absolute path to the SQLite database file.
func (s *Store) DBPath() string { return s.path }

// IsInitialized reports whether a password hash has been stored.
func (s *Store) IsInitialized(ctx context.Context) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, sqlCountCredentials).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// SetPassword hashes plain with bcrypt (cost 12) and stores (or replaces) it.
// The plaintext is never written to disk.
func (s *Store) SetPassword(ctx context.Context, plain string) error {
	if len([]rune(plain)) == 0 {
		return errors.New("password must not be empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, sqlUpsertHash, string(hash))
	return err
}

// VerifyPassword returns true iff plain matches the stored bcrypt hash.
// Returns (false, nil) when no password has been set yet.
func (s *Store) VerifyPassword(ctx context.Context, plain string) (bool, error) {
	var hash string
	err := s.db.QueryRowContext(ctx, sqlSelectHash).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return err == nil, err
}