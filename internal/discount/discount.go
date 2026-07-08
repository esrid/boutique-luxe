// Package discount owns discount codes: admin CRUD and the lookup/apply
// path checkout uses.
package discount

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound = errors.New("discount: not found")
	ErrInvalid  = errors.New("discount: code is inactive or expired")
)

type Type string

const (
	TypePercent Type = "percent"
	TypeFixed   Type = "fixed"
)

type Code struct {
	ID        int64
	Code      string
	Type      Type
	Value     int64 // percent: 1-100; fixed: cents
	Active    bool
	ExpiresAt *time.Time
}

// ComputeDiscountCents returns how much to take off subtotalCents. Never
// discounts below zero, no matter how the code is configured.
func (c Code) ComputeDiscountCents(subtotalCents int64) int64 {
	var d int64
	switch c.Type {
	case TypePercent:
		d = subtotalCents * c.Value / 100
	case TypeFixed:
		d = c.Value
	}
	if d > subtotalCents {
		d = subtotalCents
	}
	if d < 0 {
		d = 0
	}
	return d
}

func (c Code) IsValid(now time.Time) bool {
	if !c.Active {
		return false
	}
	if c.ExpiresAt != nil && now.After(*c.ExpiresAt) {
		return false
	}
	return true
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type Params struct {
	Code      string
	Type      Type
	Value     int64
	Active    bool
	ExpiresAt *time.Time
}

func (s *Store) Create(ctx context.Context, p Params) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO discount_codes (code, type, value, active, expires_at) VALUES (?, ?, ?, ?, ?)`,
		p.Code, string(p.Type), p.Value, boolToInt(p.Active), formatExpiry(p.ExpiresAt))
	if err != nil {
		return 0, fmt.Errorf("create discount code: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) Update(ctx context.Context, id int64, p Params) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE discount_codes SET code = ?, type = ?, value = ?, active = ?, expires_at = ? WHERE id = ?`,
		p.Code, string(p.Type), p.Value, boolToInt(p.Active), formatExpiry(p.ExpiresAt), id)
	if err != nil {
		return fmt.Errorf("update discount code: %w", err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM discount_codes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete discount code: %w", err)
	}
	return nil
}

func (s *Store) List(ctx context.Context) ([]Code, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, code, type, value, active, expires_at FROM discount_codes ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list discount codes: %w", err)
	}
	defer rows.Close()

	var out []Code
	for rows.Next() {
		c, err := scanCode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetByID(ctx context.Context, id int64) (Code, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, code, type, value, active, expires_at FROM discount_codes WHERE id = ?`, id)
	c, err := scanCode(row)
	if err == sql.ErrNoRows {
		return Code{}, ErrNotFound
	}
	if err != nil {
		return Code{}, err
	}
	return c, nil
}

// FindActiveByCode looks up a code (case-insensitive) and checks it's
// currently usable. Returns ErrNotFound if no such code exists at all, or
// ErrInvalid if it exists but is inactive/expired — callers should show
// the customer the same "invalid code" message either way.
func (s *Store) FindActiveByCode(ctx context.Context, code string) (Code, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, code, type, value, active, expires_at FROM discount_codes WHERE lower(code) = lower(?)`, code)
	c, err := scanCode(row)
	if err == sql.ErrNoRows {
		return Code{}, ErrNotFound
	}
	if err != nil {
		return Code{}, err
	}
	if !c.IsValid(time.Now()) {
		return Code{}, ErrInvalid
	}
	return c, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCode(row scanner) (Code, error) {
	var c Code
	var typ string
	var active int
	var expiresAt sql.NullString
	if err := row.Scan(&c.ID, &c.Code, &typ, &c.Value, &active, &expiresAt); err != nil {
		return Code{}, fmt.Errorf("scan discount code: %w", err)
	}
	c.Type = Type(typ)
	c.Active = active != 0
	if expiresAt.Valid {
		t, err := time.Parse("2006-01-02 15:04:05", expiresAt.String)
		if err == nil {
			c.ExpiresAt = &t
		}
	}
	return c, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func formatExpiry(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}
