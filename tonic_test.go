package tonic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

const redisURL = "redis://:my_password@localhost:6379/0?protocol=3"

func setupTest(t *testing.T) (*Limiter, *redis.Client) {
	opts, err := redis.ParseURL(redisURL)
	assert.NoError(t, err)
	mockRedis := redis.NewClient(opts)
	limiter := &Limiter{redisClient: mockRedis}

	// Clear Redis before test
	ctx := context.Background()
	err = mockRedis.FlushAll(ctx).Err()
	assert.NoError(t, err)

	return limiter, mockRedis
}

func TestNew(t *testing.T) {
	limiter, err := New(redisURL)
	assert.NoError(t, err)
	assert.NotNil(t, limiter)
	assert.NotNil(t, limiter.redisClient)
}

func TestParseLimit(t *testing.T) {
	limiter, _ := setupTest(t)

	tests := []struct {
		name        string
		input       string
		expected    *Limit
		expectError bool
	}{
		{"Valid second", "10/second", &Limit{10, time.Second}, false},
		{"Valid minute", "5/minute", &Limit{5, time.Minute}, false},
		{"Valid hour", "100/hour", &Limit{100, time.Hour}, false},
		{"Invalid format", "10", nil, true},
		{"Invalid quota", "abc/second", nil, true},
		{"Invalid duration", "10/day", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := limiter.parseLimit(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestLimitMiddleware(t *testing.T) {
	limiter, _ := setupTest(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", limiter.Limit("2/minute"), func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	makeRequest := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
		return w
	}

	for i := 0; i < 2; i++ {
		w := makeRequest()
		assert.Equal(t, http.StatusOK, w.Code)
	}

	w := makeRequest()
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	time.Sleep(time.Minute)

	w = makeRequest()
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLimitMiddlewareWithDifferentLimits(t *testing.T) {
	limiter, _ := setupTest(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/fast", limiter.Limit("100/second"), func(c *gin.Context) {
		c.String(http.StatusOK, "Fast")
	})
	router.GET("/slow", limiter.Limit("2/minute"), func(c *gin.Context) {
		c.String(http.StatusOK, "Slow")
	})

	makeRequest := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		router.ServeHTTP(w, req)
		return w
	}

	for i := 0; i < 100; i++ {
		w := makeRequest("/fast")
		assert.Equal(t, http.StatusOK, w.Code)
	}
	w := makeRequest("/fast")
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	for i := 0; i < 2; i++ {
		w := makeRequest("/slow")
		assert.Equal(t, http.StatusOK, w.Code)
	}
	w = makeRequest("/slow")
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}