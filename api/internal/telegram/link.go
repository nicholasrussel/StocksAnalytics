package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tripleboys27/stock-dashboard/api/internal/store"
)

// LinkStore redeems the one-time tokens created from the web app.
type LinkStore interface {
	UseLinkToken(ctx context.Context, token string, chatID int64) (*store.User, error)
}

// Sender delivers a reply to a Telegram chat.
type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
}

// HandleMessage processes an incoming message from either webhook or polling.
// Anyone can message the bot, so replies never include internal error details.
func HandleMessage(ctx context.Context, links LinkStore, sender Sender, msg *Message) {
	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID

	if !strings.HasPrefix(text, "/start") {
		return
	}

	parts := strings.Fields(text)
	if len(parts) < 2 {
		_ = sender.SendMessage(ctx, chatID, "Gunakan link dari aplikasi web untuk menghubungkan akun kamu.")
		return
	}

	user, err := links.UseLinkToken(ctx, parts[1], chatID)
	if err != nil {
		var reply string
		switch {
		case errors.Is(err, store.ErrTokenNotFound):
			reply = "❌ Token tidak ditemukan. Silakan buat link baru dari aplikasi."
		case errors.Is(err, store.ErrTokenUsed):
			reply = "❌ Token sudah pernah digunakan."
		case errors.Is(err, store.ErrTokenExpired):
			reply = "❌ Token sudah kedaluwarsa. Silakan buat link baru dari aplikasi."
		default:
			reply = "❌ Terjadi kesalahan. Silakan coba lagi nanti."
		}
		_ = sender.SendMessage(ctx, chatID, reply)
		slog.Warn("link token error", "chat_id", chatID, "err", err)
		return
	}

	name := user.Email
	if user.Name != nil && *user.Name != "" {
		name = *user.Name
	}
	reply := fmt.Sprintf("✅ Berhasil terhubung!\n\nHai <b>%s</b>, akun Telegram kamu sudah terhubung ke Stock Research Dashboard.\nKamu akan mendapat notifikasi berita penting sesuai watchlist.", name)
	_ = sender.SendMessage(ctx, chatID, reply)
	slog.Info("telegram linked", "user", user.Email, "chat_id", chatID)
}
