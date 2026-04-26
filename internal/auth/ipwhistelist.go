package auth

import (
	"net"
	"net/http"
	"os"
	"strings"

	"log/slog"

	"github.com/gin-gonic/gin"
)

// IPWhitelist returns a middleware that restricts webhook access by IP.
// Set ALLOWED_IPS=196.47.12.0/24,196.47.13.0/24 in .env
// If ALLOWED_IPS is empty, all IPs are allowed.
func IPWhitelist() gin.HandlerFunc {
	raw := os.Getenv("ALLOWED_IPS")

	// No env var set — allow all
	if raw == "" {
		slog.Info("IP whitelisting disabled — set ALLOWED_IPS to enable")
		return func(c *gin.Context) { c.Next() }
	}

	var networks []*net.IPNet
	for _, cidr := range strings.Split(raw, ",") {
		cidr = strings.TrimSpace(cidr)
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			slog.Warn("invalid CIDR in ALLOWED_IPS — skipping",
				"cidr", cidr,
				"error", err.Error(),
			)
			continue
		}
		networks = append(networks, network)
	}

	slog.Info("IP whitelisting enabled", "ranges", raw)

	return func(c *gin.Context) {
		clientIP := net.ParseIP(c.ClientIP())
		if clientIP == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "could not determine client IP",
			})
			return
		}

		for _, network := range networks {
			if network.Contains(clientIP) {
				c.Next()
				return
			}
		}

		slog.Warn("webhook request from non-whitelisted IP",
			"ip", clientIP.String(),
		)
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "IP not allowed",
		})
	}
}
