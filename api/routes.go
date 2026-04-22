package api

import (
	"net/http"

	"github.com/Brown-Moses/paykit/internal/auth"
	"github.com/Brown-Moses/paykit/internal/storage"
	"github.com/Brown-Moses/paykit/internal/webhook"
)

func NewRouter(verifier *auth.Verifier, store *storage.Store) http.Handler {
	mux := http.NewServeMux()

	webHook := webhook.NewHandler(verifier, store)
	mux.HandleFunc("POST /webhook/momo", webHook.HandlerMoMoWebhook)

	return mux
}
