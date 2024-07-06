package tonic

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	redisClient *redis.Client
}

type Limit struct {
	quota    int
	duration time.Duration
}

func New(redisURL string) (*Limiter, error) {

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &Limiter{redisClient: client}, nil
}

func (l *Limiter) Limit(limit string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		parsedLimit, err := l.parseLimit(limit)
		if err != nil {
			panic(err)
		}
		key := fmt.Sprintf("path:%v;ip:%v;quota:%v;duration:%v", ctx.FullPath(), ctx.ClientIP(), parsedLimit.quota, parsedLimit.duration)
		log.Printf("key: %v\n", key)
		val, err := l.redisClient.Get(ctx, key).Result()

		internalErr := fmt.Errorf("something went wrong")
		if err == redis.Nil {
			_, err = l.redisClient.Set(ctx, key, 1, parsedLimit.duration).Result()
			if err != nil {
				log.Printf("Error setting Redis key: %v\n", err)
				ctx.AbortWithError(http.StatusInternalServerError, internalErr)
				return
			}
		} else if err != nil {
			log.Printf("Redis error: %v\n", err)
			ctx.AbortWithError(http.StatusInternalServerError, internalErr)
			return
		} else {
			count, err := strconv.Atoi(val)
			if err != nil {
				log.Printf("Error parsing count: %v\n", err)
				ctx.AbortWithError(http.StatusInternalServerError, internalErr)
				return
			}
			if count < parsedLimit.quota {
				_, err = l.redisClient.Incr(ctx, key).Result()
				if err != nil {
					log.Printf("Error incrementing count: %v\n", err)
					ctx.AbortWithError(http.StatusInternalServerError, internalErr)
					return
				}
			} else {
				log.Printf("Rate limit reached: %v\n", key)
				ctx.AbortWithError(http.StatusTooManyRequests, fmt.Errorf("rate limit reached"))
				return
			}
		}
		ctx.Next()
	}
}

func (l *Limiter) parseLimit(limit string) (*Limit, error) {
	items := strings.Split(limit, "/")

	limitErr := fmt.Errorf("invalid limit string: %v; expected format: <quota>/<duration>; example: 5/minute", items)

	if len(items) != 2 {
		return nil, limitErr
	}

	quota, err := strconv.Atoi(items[0])
	if err != nil {
		return nil, limitErr
	}

	duration := items[1]
	switch duration {
	case "second":
		return &Limit{quota, time.Second}, nil
	case "minute":
		return &Limit{quota, time.Minute}, nil
	case "hour":
		return &Limit{quota, time.Hour}, nil
	default:
		return nil, limitErr
	}
}