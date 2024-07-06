# tonic

Looking for a simple rate limiter for the Gin web framework? 

tonic is a distributed, fixed-window rate limiter for Gin, using Redis for storage.

## Features
- Easy integration with Gin
- Flexible rate limiting based on request path and client IP
- Supports limits per second, minute, and hour
- Distributed rate limiting using Redis

## Installation
To install tonic, use the following command:
```bash
go get -u github.com/rossheat/tonic
```

## Usage
Here's an example of how to use tonic in your Gin application:
```go
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
```
You can find the complete version of this example [here](./example//example.go).

## Configuration
The `New` function creates a new Limiter instance. It requires a Redis URL string:
```go
limiter, err := tonic.New(redisURL)
```
Note that you should create only one instance of the limiter. 

## Rate Limiting
Use the `Limit` method to apply rate limiting to routes or groups. The limit is specified as a string in the format `"<quota>/<duration>"`, where duration can be "second", "minute", or "hour":
```go
router.GET("/path", limiter.Limit("100/minute"), HandlerFunc)
```

## Error Handling
tonic will automatically abort the request with a `429 Too Many Requests` status code when the rate limit is exceeded. Other errors (e.g., Redis connection issues) will result in a `500 Internal Server Error`.

## Testing
To run the tests for tonic, use the following command:
```bash
go test ./...
```
Ensure you have a Redis instance running and accessible for the tests to pass.

## Contributing
At this time, we are not accepting contributions to tonic. We appreciate your interest, but the project is currently maintained internally.

## License
This project is licensed under the [MIT License](LICENSE.md).