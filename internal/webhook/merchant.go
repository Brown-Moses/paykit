package webhook

import (
	"net/http"

	"github.com/Brown-Moses/paykit/internal/auth"
	"github.com/Brown-Moses/paykit/internal/storage"
	"github.com/gin-gonic/gin"
)

// CreateMerchant godoc
// @Summary      Register a new merchant
// @Description  Creates a merchant account and returns a unique API key. Store the key safely — it will not be shown again.
// @Tags         Merchants
// @Accept       json
// @Produce      json
// @Param        request  body      object{name=string,webhook_url=string}  true  "Merchant details"
// @Success      201      {object}  object{id=int,name=string,api_key=string,webhook_url=string,message=string}
// @Failure      400      {object}  object{error=string}
// @Failure      500      {object}  object{error=string}
// @Router       /merchants [post]

// CreateMerchant registers a new merchant and returns their API key
func (h *Handler) CreateMerchant(c *gin.Context) {
	var body struct {
		Name       string `json:"name"       binding:"required"`
		WebhookURL string `json:"webhook_url"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey, err := auth.GenerateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate API key"})
		return
	}

	id, err := h.store.CreateMerchant(storage.Merchant{
		Name:       body.Name,
		APIKey:     apiKey,
		WebhookURL: body.WebhookURL,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create merchant"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          id,
		"name":        body.Name,
		"api_key":     apiKey,
		"webhook_url": body.WebhookURL,
		"message":     "store this api_key safely — it will not be shown again",
	})
}
