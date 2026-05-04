//go:build chaos

package chaos_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zhaba1337228/GuardGo"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRedisChaosFailOpenSurvivesContainerKill(t *testing.T) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = container.Terminate(ctx) }()

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatal(err)
	}

	addr := fmt.Sprintf("%s:%s", host, port.Port())
	rdb := redis.NewClient(&redis.Options{Addr: addr, MaxRetries: 0})
	engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
		Redis:       rdb,
		MaxRequests: 1000000,
		Window:      time.Second,
		FailOpen:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	var allowed atomic.Int64
	var wg sync.WaitGroup
	stop := make(chan struct{})

	worker := func(id int) {
		defer wg.Done()
		req, _ := http.NewRequest(http.MethodGet, "http://example.com/chaos", nil)
		req.RemoteAddr = fmt.Sprintf("100.0.0.%d:9000", id)
		for {
			select {
			case <-stop:
				return
			default:
				d := engine.Process(ctx, req)
				if d.Allowed {
					allowed.Add(1)
				}
			}
		}
	}

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go worker(i + 1)
	}

	time.Sleep(500 * time.Millisecond)
	if err := container.Stop(ctx, nil); err != nil {
		t.Fatal(err)
	}

	time.Sleep(1200 * time.Millisecond)
	close(stop)
	wg.Wait()

	if allowed.Load() == 0 {
		t.Fatalf("expected allowed requests even during redis outage in fail-open mode")
	}
}
