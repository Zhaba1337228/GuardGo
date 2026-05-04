package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	guardgo "github.com/Zhaba1337228/GuardGo"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	cfg := guardgo.NewConfig(rdb, 300, time.Second)
	cfg.Reputation.Enabled = true
	cfg.Bloom.Enabled = true

	engine := guardgo.New(cfg)
	defer engine.Close()

	router := gin.New()
	router.Use(guardgo.Gin(engine))
	router.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })

	_ = router.Run(":8080")
}
