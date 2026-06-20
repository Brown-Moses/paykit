package auth

import (
	"net/http"
	"strings"

	"github.com/Brown-Moses/paykit/internal/storage"
	"github.com/gin-gonic/gin"
)

const MerchantKey = "merchant"

// requireAPIKey validates bearer token and injects merchent into context
func RequireAPIKey(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing Authorization header",
			})
			return
		}

		//expect: bearer pk_live_xxx
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid Authorization format - use: Bearer pk_live_xxx",
			})
			return
		}

		apiKey := parts[1]
		merchant, err := store.GetMerchantByAPIKey(apiKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or inactive API Key",
			})
			return
		}

		//inject merchant into context
		c.Set(MerchantKey, merchant)
		c.Next()
	}
}

// extract the merchant from gin
func MerchantFrom(c *gin.Context) *storage.Merchant {
	m, _ := c.Get(MerchantKey)
	merchant, _ := m.(*storage.Merchant)
	return merchant
}
