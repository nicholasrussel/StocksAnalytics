package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tripleboys27/stock-dashboard/api/internal/telegram"
)

func init() { gin.SetMode(gin.TestMode) }

func TestInternalKeyMiddleware(t *testing.T) {
	tests := []struct {
		name, key, header string
		sendHeader        bool
		want              int
	}{
		{"empty key rejects missing header", "", "", false, http.StatusUnauthorized},
		{"empty key rejects empty header", "", "", true, http.StatusUnauthorized},
		{"missing header", "secret", "", false, http.StatusUnauthorized},
		{"wrong key", "secret", "nope", true, http.StatusUnauthorized},
		{"correct key", "secret", "secret", true, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("INTERNAL_API_KEY", tt.key)
			r := gin.New()
			r.GET("/x", internalKeyMiddleware(), func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tt.sendHeader {
				req.Header.Set("X-Internal-Key", tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestTelegramWebhook(t *testing.T) {
	const message = `{"update_id":1,"message":{"text":"/start abc","chat":{"id":42}}}`
	tests := []struct {
		name, configured, header, body string
		wantStatus                     int
		wantHandled                    bool
	}{
		{"valid", "s3cret", "s3cret", message, http.StatusOK, true},
		{"update without message", "s3cret", "s3cret", `{"update_id":2}`, http.StatusOK, false},
		{"wrong secret", "s3cret", "nope", message, http.StatusUnauthorized, false},
		{"missing secret", "s3cret", "", message, http.StatusUnauthorized, false},
		{"unconfigured secret", "", "", message, http.StatusUnauthorized, false},
		{"invalid json", "s3cret", "s3cret", `{`, http.StatusBadRequest, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handled := false
			r := gin.New()
			r.POST("/telegram/webhook", telegramWebhookHandler(tt.configured, func(_ context.Context, m *telegram.Message) {
				handled = m.Chat.ID == 42 && m.Text == "/start abc"
			}))

			req := httptest.NewRequest(http.MethodPost, "/telegram/webhook", strings.NewReader(tt.body))
			if tt.header != "" {
				req.Header.Set(telegramSecretHeader, tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus || handled != tt.wantHandled {
				t.Fatalf("status = %d handled = %v, want %d %v", w.Code, handled, tt.wantStatus, tt.wantHandled)
			}
		})
	}
}
