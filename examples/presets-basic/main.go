package main

import (
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	guardgo "github.com/Zhaba1337228/GuardGo"
	"github.com/Zhaba1337228/GuardGo/rules"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	cfg := guardgo.NewConfig(rdb, 300, time.Second)
	cfg.Reputation.Enabled = true
	cfg.Reputation.WarningLevel = 60
	cfg.Reputation.Threshold = 100
	rules.ApplyDefaultSecurityPresets(&cfg)

	engine := guardgo.New(cfg)
	defer engine.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	_ = http.ListenAndServe(":8080", engine.Middleware(mux))
}
