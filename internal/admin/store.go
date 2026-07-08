// Package admin holds the back-office: login, and (in later slices)
// catalog/order/inventory management screens.
package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrInvalidCredentials = errors.New("admin: invalid credentials")

type User struct {
	ID    int64
	Email string
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// FindByEmail returns the user and password hash for login verification.
func (s *Store) FindByEmail(ctx context.Context, email string) (User, string, error) {
	var u User
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT id, email, password_hash FROM admin_users WHERE email = ?`, email).
		Scan(&u.ID, &u.Email, &hash)
	if err == sql.ErrNoRows {
		return User{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return User{}, "", fmt.Errorf("find admin: %w", err)
	}
	return u, hash, nil
}

func (s *Store) CreateSession(ctx context.Context, adminID int64, tokenHash string, ttl time.Duration) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO admin_sessions (admin_id, token_hash, expires_at) VALUES (?, ?, datetime('now', ?))`,
		adminID, tokenHash, fmt.Sprintf("+%d seconds", int(ttl.Seconds())))
	if err != nil {
		return fmt.Errorf("create admin session: %w", err)
	}
	return nil
}

// FindSession returns the admin user for a valid, unexpired session token
// hash.
func (s *Store) FindSession(ctx context.Context, tokenHash string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `
		SELECT admin_users.id, admin_users.email
		FROM admin_sessions
		JOIN admin_users ON admin_users.id = admin_sessions.admin_id
		WHERE admin_sessions.token_hash = ? AND admin_sessions.expires_at > datetime('now')`,
		tokenHash,
	).Scan(&u.ID, &u.Email)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE token_hash = ?`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete admin session: %w", err)
	}
	return nil
}
