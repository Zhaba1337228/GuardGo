package main

import (
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"guardgo"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})

	cfg := guardgo.NewConfig(rdb, 500, time.Second)
	cfg.FailOpen = true
	cfg.Reputation.Enabled = true
	cfg.Reputation.WarningLevel = 60
	cfg.Reputation.Threshold = 100

	engine := guardgo.New(cfg)
	defer engine.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	_ = http.ListenAndServe(":8080", engine.Middleware(mux))
}
