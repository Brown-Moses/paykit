package api

import (
	"github.com/Brown-Moses/paykit/internal/auth"
	"github.com/Brown-Moses/paykit/internal/storage"
	webhooks "github.com/Brown-Moses/paykit/internal/webhook"
	"github.com/gin-gonic/gin"
)

func NewRouter(verifier *auth.Verifier, store *storage.Store) *gin.Engine {
	r := gin.Default()

	webHook := webhooks.NewHandler(verifier, store)

	r.POST("/webhook/momo", webHook.HandleMoMoWebhook)
	r.GET("/transactions/:id", webHook.GetTransaction)

	return r
}
