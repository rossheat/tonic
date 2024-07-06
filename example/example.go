package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rossheat/tonic"
)

func main() {

	redisUrl := "redis://:my_password@localhost:6379/0?protocol=3"

	limiter, err := tonic.New(redisUrl)

	if err != nil {
		panic(err)
	}

	router := gin.Default()

	router.GET("/fast", limiter.Limit("100/second"), Fast)

	router.GET("/slow", limiter.Limit("5/minute"), Slow)

	// These routes each have a rate limit of 100 requests per hour.
	slowGroup := router.Group("/slow-group", limiter.Limit("100/hour"))
	{
		slowGroup.GET("one", SlowOne)
		slowGroup.GET("two", SlowTwo)
	}

	router.Run(":8080")
}

func Fast(ctx *gin.Context) {
	ctx.String(http.StatusOK, "Fast")
}

func Slow(ctx *gin.Context) {
	ctx.String(http.StatusOK, "Slow")
}

func SlowOne(ctx *gin.Context) {
	ctx.String(http.StatusOK, "SlowOne")
}

func SlowTwo(ctx *gin.Context) {
	ctx.String(http.StatusOK, "SlowTwo")
}
