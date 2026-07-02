package auth

import (
	"fmt"
	"net/http"

	"github.com/Brown-Moses/paykit/internal/storage"
	"github.com/gin-gonic/gin"
)

// TierEnforcement blocks webhook processing when a merchant hits their limit
// and appends a warning header at 80% usage.
func TierEnforcement(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		merchantIDStr := c.Param("merchant_id")
		var merchantID int64
		if _, err := fmt.Sscanf(merchantIDStr, "%d", &merchantID); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid merchant_id in URL path"})
			return
		}

		merchant, err := store.GetMerchantByID(merchantID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "merchant not found or inactive"})
			return
		}

		// Block if the merchant has hit or exceeded their limit
		if merchant.CurrentMonthCalls >= merchant.MaxMonthlyCalls {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "monthly webhook limit exceeded",
				"limit": merchant.MaxMonthlyCalls,
				"usage": merchant.CurrentMonthCalls,
			})
			return
		}

		// Append warning header at 80% limit usage
		if float64(merchant.CurrentMonthCalls) >= 0.8*float64(merchant.MaxMonthlyCalls) {
			c.Header("X-Quota-Warning", fmt.Sprintf("Usage is at %d/%d (%d%%)", 
				merchant.CurrentMonthCalls, 
				merchant.MaxMonthlyCalls, 
				int(float64(merchant.CurrentMonthCalls)/float64(merchant.MaxMonthlyCalls)*100),
			))
		}

		// Inject the merchant so handlers don't need to fetch it again
		c.Set(MerchantKey, merchant)
		c.Next()
	}
}
