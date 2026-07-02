package webhook

import (
	"fmt"
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

// Login godoc
// @Summary      Authenticate merchant
// @Description  Logs in a merchant using their API key and returns current status and usage.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      object{api_key=string}  true  "Credentials"
// @Success      200      {object}  object{id=int,name=string,plan_type=string,current_month_calls=int,max_monthly_calls=int}
// @Failure      400      {object}  object{error=string}
// @Failure      401      {object}  object{error=string}
// @Router       /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var body struct {
		APIKey string `json:"api_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key is required"})
		return
	}

	merchant, err := h.store.GetMerchantByAPIKey(body.APIKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API Key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  merchant.ID,
		"name":                merchant.Name,
		"plan_type":           merchant.PlanType,
		"current_month_calls": merchant.CurrentMonthCalls,
		"max_monthly_calls":   merchant.MaxMonthlyCalls,
		"webhook_url":         merchant.WebhookURL,
	})
}

// UpdateWebhookURL godoc
// @Summary      Update merchant webhook URL
// @Description  Updates the destination URL where PayKit forwards verified notifications.
// @Tags         Merchants
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      object{webhook_url=string}  true  "Webhook URL details"
// @Success      200      {object}  object{message=string,webhook_url=string}
// @Failure      400      {object}  object{error=string}
// @Failure      401      {object}  object{error=string}
// @Failure      500      {object}  object{error=string}
// @Router       /merchants/webhook-url [put]
func (h *Handler) UpdateWebhookURL(c *gin.Context) {
	merchant := auth.MerchantFrom(c)
	var body struct {
		WebhookURL string `json:"webhook_url" binding:"required,url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a valid webhook_url is required"})
		return
	}

	if err := h.store.UpdateMerchantWebhookURL(merchant.ID, body.WebhookURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update webhook URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "webhook URL updated successfully",
		"webhook_url": body.WebhookURL,
	})
}

// AdminListMerchants lists all merchants for operator dashboard
func (h *Handler) AdminListMerchants(c *gin.Context) {
	merchants, err := h.store.ListAllMerchants()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch merchants"})
		return
	}
	c.JSON(http.StatusOK, merchants)
}

// AdminToggleMerchant activates/deactivates a merchant
func (h *Handler) AdminToggleMerchant(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid merchant ID"})
		return
	}

	var body struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "active parameter required"})
		return
	}

	if err := h.store.SetMerchantActive(id, body.Active); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update merchant status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "merchant status updated", "active": body.Active})
}

// AdminUpdateMerchantQuota updates merchant plan and max call quota
func (h *Handler) AdminUpdateMerchantQuota(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid merchant ID"})
		return
	}

	var body struct {
		PlanType        string `json:"plan_type" binding:"required"`
		MaxMonthlyCalls int    `json:"max_monthly_calls" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan_type or max_monthly_calls parameters"})
		return
	}

	if err := h.store.UpdateMerchantQuota(id, body.PlanType, body.MaxMonthlyCalls); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update merchant plan quota"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "merchant quota updated",
		"plan_type":         body.PlanType,
		"max_monthly_calls": body.MaxMonthlyCalls,
	})
}
