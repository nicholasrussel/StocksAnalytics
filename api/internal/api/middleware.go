package api

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tripleboys27/stock-dashboard/api/internal/store"
)

func internalKeyMiddleware() gin.HandlerFunc {
	key := strings.TrimSpace(os.Getenv("INTERNAL_API_KEY"))
	return func(c *gin.Context) {
		incoming := strings.TrimSpace(c.GetHeader("X-Internal-Key"))
		if key == "" || subtle.ConstantTimeCompare([]byte(incoming), []byte(key)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

func userMiddleware(db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		email := strings.ToLower(strings.TrimSpace(c.GetHeader("X-User-Email")))
		if email == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing X-User-Email"})
			return
		}
		user, err := db.GetUserByEmail(c.Request.Context(), email)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		c.Set("user", user)
		c.Next()
	}
}
