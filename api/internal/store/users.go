package store

import (
	"context"
	"errors"
	"time"
)

type User struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	Name           *string    `json:"name"`
	Role           string     `json:"role"`
	TelegramChatID *int64     `json:"telegram_chat_id"`
	MinImportance  int        `json:"min_importance"`
	DigestEnabled  bool       `json:"digest_enabled"`
	CreatedAt      time.Time  `json:"created_at"`
	LastLoginAt    *time.Time `json:"last_login_at"`
}

var ErrNotAllowed = errors.New("email not in allowlist")

func (db *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := db.QueryRow(ctx, `
		SELECT u.id, u.email, u.name, ae.role,
		       u.telegram_chat_id, u.min_importance, u.digest_enabled,
		       u.created_at, u.last_login_at
		FROM users u
		JOIN allowed_emails ae ON ae.email = u.email
		WHERE u.email = $1
	`, email).Scan(
		&u.ID, &u.Email, &u.Name, &u.Role,
		&u.TelegramChatID, &u.MinImportance, &u.DigestEnabled,
		&u.CreatedAt, &u.LastLoginAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) GetOrCreateUser(ctx context.Context, email, name string) (*User, error) {
	// Check allowlist
	var role string
	var isActive bool
	err := db.QueryRow(ctx,
		`SELECT role, is_active FROM allowed_emails WHERE email = $1`, email,
	).Scan(&role, &isActive)
	if err != nil || !isActive {
		return nil, ErrNotAllowed
	}

	// Upsert into users
	var u User
	var namePtr *string
	if name != "" {
		namePtr = &name
	}
	now := time.Now()
	err = db.QueryRow(ctx, `
		INSERT INTO users (email, name, last_login_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE
		  SET name = COALESCE($2, users.name),
		      last_login_at = $3
		RETURNING id, email, name, $4::text as role,
		          telegram_chat_id, min_importance, digest_enabled,
		          created_at, last_login_at
	`, email, namePtr, now, role,
	).Scan(
		&u.ID, &u.Email, &u.Name, &u.Role,
		&u.TelegramChatID, &u.MinImportance, &u.DigestEnabled,
		&u.CreatedAt, &u.LastLoginAt,
	)
	return &u, err
}
