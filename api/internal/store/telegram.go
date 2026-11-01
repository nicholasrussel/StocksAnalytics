package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var ErrTokenNotFound = errors.New("token not found")
var ErrTokenUsed = errors.New("token already used")
var ErrTokenExpired = errors.New("token expired")

func (db *DB) CreateLinkToken(ctx context.Context, userID string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := db.Exec(ctx, `
		INSERT INTO telegram_link_tokens (token, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, token, userID, expiresAt)
	return token, err
}

func (db *DB) UseLinkToken(ctx context.Context, token string, chatID int64) (*User, error) {
	var (
		userID    string
		expiresAt time.Time
		usedAt    *time.Time
	)
	err := db.QueryRow(ctx, `
		SELECT user_id, expires_at, used_at
		FROM telegram_link_tokens WHERE token = $1
	`, token).Scan(&userID, &expiresAt, &usedAt)
	if err != nil {
		return nil, ErrTokenNotFound
	}
	if usedAt != nil {
		return nil, ErrTokenUsed
	}
	if time.Now().After(expiresAt) {
		return nil, ErrTokenExpired
	}

	_, err = db.Exec(ctx, `UPDATE telegram_link_tokens SET used_at = now() WHERE token = $1`, token)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(ctx, `UPDATE users SET telegram_chat_id = $1 WHERE id = $2`, chatID, userID)
	if err != nil {
		return nil, err
	}

	return db.GetUserByID(ctx, userID)
}

func (db *DB) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := db.QueryRow(ctx, `
		SELECT u.id, u.email, u.name, ae.role,
		       u.telegram_chat_id, u.min_importance, u.digest_enabled,
		       u.created_at, u.last_login_at
		FROM users u
		JOIN allowed_emails ae ON ae.email = u.email
		WHERE u.id = $1
	`, id).Scan(
		&u.ID, &u.Email, &u.Name, &u.Role,
		&u.TelegramChatID, &u.MinImportance, &u.DigestEnabled,
		&u.CreatedAt, &u.LastLoginAt,
	)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &u, nil
}
