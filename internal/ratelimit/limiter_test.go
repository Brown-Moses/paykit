package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Brown-Moses/paykit/internal/auth"
	"github.com/Brown-Moses/paykit/internal/storage"
	"github.com/gin-gonic/gin"
)

func TestRateLimiter_Allow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Registry with rate 2.0 RPS, burst 2
	registry := NewRegistry(2.0, 2)
	middleware := registry.Middleware()

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Make 2 requests immediately (should be allowed due to burst of 2)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d on request %d", w.Code, i+1)
		}
	}

	// 3rd request should fail with 429
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}
}

func TestRateLimiter_MerchantScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Registry with 1 RPS, burst 1
	registry := NewRegistry(1.0, 1)
	middleware := registry.Middleware()

	router := gin.New()
	// Middleware that sets merchant in context based on header for testing
	router.Use(func(c *gin.Context) {
		mIDStr := c.GetHeader("X-Test-Merchant-ID")
		if mIDStr != "" {
			id, _ := strconv.ParseInt(mIDStr, 10, 64)
			c.Set(auth.MerchantKey, &storage.Merchant{ID: id})
		}
		c.Next()
	})
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Merchant 1 request (succeeds)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.Header.Set("X-Test-Merchant-ID", "1")
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("merchant 1: expected 200, got %d", w1.Code)
	}

	// Merchant 1 request 2 (exceeds limit -> 429)
	w1_2 := httptest.NewRecorder()
	req1_2, _ := http.NewRequest("GET", "/test", nil)
	req1_2.Header.Set("X-Test-Merchant-ID", "1")
	router.ServeHTTP(w1_2, req1_2)
	if w1_2.Code != http.StatusTooManyRequests {
		t.Errorf("merchant 1: expected 429 on second request, got %d", w1_2.Code)
	}

	// Merchant 2 request (should succeed because merchant 2 has their own bucket)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Test-Merchant-ID", "2")
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("merchant 2: expected 200, got %d", w2.Code)
	}
}

func TestRateLimiter_ParamScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Registry with 1 RPS, burst 1
	registry := NewRegistry(1.0, 1)
	middleware := registry.Middleware()

	router := gin.New()
	router.Use(middleware)
	router.GET("/webhook/:merchant_id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Merchant 1 param request (succeeds)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/webhook/1", nil)
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("param merchant 1: expected 200, got %d", w1.Code)
	}

	// Merchant 1 param request 2 (fails -> 429)
	w1_2 := httptest.NewRecorder()
	req1_2, _ := http.NewRequest("GET", "/webhook/1", nil)
	router.ServeHTTP(w1_2, req1_2)
	if w1_2.Code != http.StatusTooManyRequests {
		t.Errorf("param merchant 1: expected 429, got %d", w1_2.Code)
	}

	// Merchant 2 param request (succeeds)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/webhook/2", nil)
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("param merchant 2: expected 200, got %d", w2.Code)
	}
}
