package api

import (
	"github.com/Brown-Moses/paykit/internal/auth"
	"github.com/Brown-Moses/paykit/internal/storage"
	webhooks "github.com/Brown-Moses/paykit/internal/webhook"
	"github.com/gin-gonic/gin"
)

func NewRouter(verifier *auth.Verifier, store *storage.Store) *gin.Engine {
	r := gin.Default()

	r.POST("/webhook/momo", webhooks.NewHandler(verifier, store).HandleMoMoWebhook)

	return r
}
