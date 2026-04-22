package webhook

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/Brown-Moses/paykit/internal/auth"
	"github.com/Brown-Moses/paykit/internal/payments"
	"github.com/Brown-Moses/paykit/internal/storage"
	"github.com/Brown-Moses/paykit/pkg/momodto"
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
	//read raw bod first(hmac)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "couldnot read request body")
		return
	}

	defer r.Body.Close()

	//Decode to get providerTxID for the replay check
	var raw momodto.WebhookPayload
	if err := json.Unmarshal(body, &raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	//verify - hmac signature + replay attack check(Must happen before trusting the payload)
	signature := r.Header.Get("X-Signature")
	if err := h.verifier.Verify(body, signature, raw.TransactionID); err != nil {
		switch {
		case errors.Is(err, auth.ErrMissingSignature):
			writeError(w, http.StatusUnauthorized, "missing signature header")
		case errors.Is(err, auth.ErrInvalidSignature):
			writeError(w, http.StatusUnauthorized, "invalid signature")
		case errors.Is(err, auth.ErrReplay):
			// Return 200 on replays — MTN will keep retrying if we return an error.
			// Silently accept and move on.
			log.Printf("replay detected for tx %s — ignoring", raw.TransactionID)
			w.WriteHeader(http.StatusOK)
		default:
			writeError(w, http.StatusInternalServerError, "verification failed")
		}
		return
	}

	//parse raw payload into internal model
	transaction, err := payments.FromWebhook(raw, body)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	//save to postgres
	if err := h.store.Insert(*transaction); err != nil {
		log.Printf("failed to insert transaction %s: %v", transaction.ProviderTxID, err)
		writeError(w, http.StatusInternalServerError, "could not save transaction")
		return
	}

	log.Printf("transaction %s recieved - external_id: %s amount: %s %s", transaction.ProviderTxID, transaction.ExternalID, transaction.Amount, transaction.Currency)

	//always return 200 quickly- MTN will retry on anything else
	w.WriteHeader(http.StatusOK)

}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
