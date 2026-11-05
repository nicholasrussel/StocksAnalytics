package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tripleboys27/stock-dashboard/api/internal/store"
)

type fakeLinks struct {
	user  *store.User
	err   error
	token string
	chat  int64
}

func (f *fakeLinks) UseLinkToken(_ context.Context, token string, chatID int64) (*store.User, error) {
	f.token, f.chat = token, chatID
	return f.user, f.err
}

type fakeSender struct{ sent []string }

func (f *fakeSender) SendMessage(_ context.Context, _ int64, text string) error {
	f.sent = append(f.sent, text)
	return nil
}

func TestHandleMessageLinksAccount(t *testing.T) {
	links := &fakeLinks{user: &store.User{Email: "a@example.com"}}
	sender := &fakeSender{}

	HandleMessage(context.Background(), links, sender, &Message{Text: "/start abc123", Chat: Chat{ID: 42}})

	if links.token != "abc123" || links.chat != 42 {
		t.Fatalf("UseLinkToken got token=%q chat=%d", links.token, links.chat)
	}
	if len(sender.sent) != 1 || !strings.Contains(sender.sent[0], "a@example.com") {
		t.Fatalf("unexpected replies: %v", sender.sent)
	}
}

func TestHandleMessageIgnoresOtherText(t *testing.T) {
	links := &fakeLinks{}
	sender := &fakeSender{}

	HandleMessage(context.Background(), links, sender, &Message{Text: "halo"})

	if links.token != "" || len(sender.sent) != 0 {
		t.Fatalf("expected no action, token=%q replies=%v", links.token, sender.sent)
	}
}

func TestHandleMessageErrorReplies(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"not found", store.ErrTokenNotFound, "tidak ditemukan"},
		{"used", store.ErrTokenUsed, "sudah pernah digunakan"},
		{"expired", store.ErrTokenExpired, "kedaluwarsa"},
		{"internal", errors.New("pq: secret-host down"), "Terjadi kesalahan"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &fakeSender{}
			HandleMessage(context.Background(), &fakeLinks{err: tt.err}, sender, &Message{Text: "/start x"})

			if len(sender.sent) != 1 || !strings.Contains(sender.sent[0], tt.want) {
				t.Fatalf("unexpected replies: %v", sender.sent)
			}
			if strings.Contains(sender.sent[0], "secret-host") {
				t.Fatalf("reply leaked internal error: %q", sender.sent[0])
			}
		})
	}
}
