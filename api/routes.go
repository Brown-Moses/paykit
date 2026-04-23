package api

import (
	"github.com/Brown-Moses/paykit/internal/auth"
	"github.com/Brown-Moses/paykit/internal/storage"
	webhooks "github.com/Brown-Moses/paykit/internal/webhook"
	"github.com/gin-gonic/gin"
)

func NewRouter(verifier *auth.Verifier, store *storage.Store) *gin.Engine {
	r := gin.Default()
	r.SetTrustedProxies(nil)

	webHook := webhooks.NewHandler(verifier, store)

	//public -no auth
	r.POST("/webhook/momo", webHook.HandleMoMoWebhook)
	r.POST("/merchants", webHook.CreateMerchant)

	//protected -requires API key
	protected := r.Group("/")
	protected.Use(auth.RequireAPIKey(store))
	{
		protected.GET("/transaction/:id", webHook.GetTransaction)
		protected.GET("/transactions/:id/deliveries", webHook.GetDeliveryLogs)
		protected.GET("/transactions", webHook.ListTransactions)
	}
	return r
}
