package api

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tripleboys27/stock-dashboard/api/internal/telegram"
)

const telegramSecretHeader = "X-Telegram-Bot-Api-Secret-Token"

// telegramWebhookHandler accepts updates only with the secret registered via
// setWebhook. Valid updates always get 200, so Telegram does not redeliver a
// message whose one-time link token has already been consumed.
func telegramWebhookHandler(secret string, handle func(context.Context, *telegram.Message)) gin.HandlerFunc {
	return func(c *gin.Context) {
		incoming := c.GetHeader(telegramSecretHeader)
		if secret == "" || subtle.ConstantTimeCompare([]byte(incoming), []byte(secret)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var update telegram.Update
		if err := c.ShouldBindJSON(&update); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid update"})
			return
		}
		if update.Message != nil {
			handle(c.Request.Context(), update.Message)
		}
		c.Status(http.StatusOK)
	}
}
