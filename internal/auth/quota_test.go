package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Brown-Moses/paykit/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTierEnforcement(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	// Connect to the database
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)

	// Create test merchant
	merchant := storage.Merchant{
		Name:            "Quota Test Merchant",
		APIKey:          "test_quota_key_unique_123",
		PlanType:        "free",
		MaxMonthlyCalls: 10,
	}

	// Clean up any stray test merchant first
	_, _ = db.Exec(ctx, "DELETE FROM merchants WHERE api_key = $1", merchant.APIKey)

	merchantID, err := store.CreateMerchant(merchant)
	if err != nil {
		t.Fatalf("failed to create test merchant: %v", err)
	}
	// Clean up after test
	defer func() {
		_, _ = db.Exec(ctx, "DELETE FROM merchants WHERE id = $1", merchantID)
	}()

	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupCalls     int
		expectedStatus int
		checkWarning   bool
	}{
		{
			name:           "Under Limit - No warning",
			setupCalls:     5,
			expectedStatus: http.StatusOK,
			checkWarning:   false,
		},
		{
			name:           "At 80% Limit - Warning",
			setupCalls:     8,
			expectedStatus: http.StatusOK,
			checkWarning:   true,
		},
		{
			name:           "At 100% Limit - Blocked",
			setupCalls:     10,
			expectedStatus: http.StatusTooManyRequests,
			checkWarning:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update merchant current calls count
			_, err := db.Exec(ctx, "UPDATE merchants SET current_month_calls = $1 WHERE id = $2", tt.setupCalls, merchantID)
			if err != nil {
				t.Fatalf("failed to setup calls: %v", err)
			}

			// Setup Gin context and recorder
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			// Setup route with middleware
			r.POST("/webhook/momo/:merchant_id", TierEnforcement(store), func(ctx *gin.Context) {
				ctx.Status(http.StatusOK)
			})

			// Create request
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/webhook/momo/%d", merchantID), nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			warning := w.Header().Get("X-Quota-Warning")
			if tt.checkWarning && warning == "" {
				t.Error("expected X-Quota-Warning header, but got none")
			}
			if !tt.checkWarning && warning != "" {
				t.Errorf("did not expect X-Quota-Warning header, but got: %s", warning)
			}
		})
	}
}
