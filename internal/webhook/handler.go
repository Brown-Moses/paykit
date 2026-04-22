package webhook

import (
	"net/http"

	"github.com/Brown-Moses/paykit/internal/auth"
	"github.com/Brown-Moses/paykit/internal/storage"
)

type Handler struct {
	verifier *auth.Verifier
	store    *storage.Store
}

func NewHandler(verifier *auth.Verifier, store *storage.Store) *Handler {
	return &Handler{
		verifier: verifier,
		store:    store,
	}
}

func (h *Handler) HandlerMoMoWebhook(w http.ResponseWriter, r *http.Request) {

}
