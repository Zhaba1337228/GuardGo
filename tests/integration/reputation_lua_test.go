package integration_test

import (
	"context"
	"testing"
	"time"

	"guardgo/internal/redislua"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestReputationLuaDynamicBackoff(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	repKey := "guardgo:rep"
	penaltyKey := "guardgo:pbox:1.1.1.1"
	blKey := "guardgo:bl:1.1.1.1"
	attemptKey := "guardgo:ban_attempts:1.1.1.1"

	call := func() int64 {
		res, err := redislua.ReputationScript.Run(
			ctx,
			rdb,
			[]string{repKey, penaltyKey, blKey, attemptKey},
			"1.1.1.1|fp",
			100.0,
			int64((15 * time.Minute).Milliseconds()),
			60.0,
			100.0,
			int64((30 * time.Minute).Milliseconds()),
			int64((24 * time.Hour).Milliseconds()),
			int64((1 * time.Minute).Milliseconds()),
			int64((10 * time.Minute).Milliseconds()),
			int64((24 * time.Hour).Milliseconds()),
		).Result()
		if err != nil {
			t.Fatal(err)
		}
		arr, ok := res.([]any)
		if !ok || len(arr) < 4 {
			t.Fatalf("unexpected lua result: %#v", res)
		}
		action, _ := arr[1].(int64)
		if action != 2 {
			t.Fatalf("expected blacklist action(2), got %d", action)
		}
		banTTL, _ := arr[2].(int64)
		return banTTL
	}

	ban1 := call()
	if ban1 != int64(time.Minute/time.Millisecond) {
		t.Fatalf("expected first ban ttl 1m, got %dms", ban1)
	}
	_ = mr.Del(blKey)

	ban2 := call()
	if ban2 != int64((10*time.Minute)/time.Millisecond) {
		t.Fatalf("expected second ban ttl 10m, got %dms", ban2)
	}
	_ = mr.Del(blKey)

	ban3 := call()
	if ban3 != int64((24*time.Hour)/time.Millisecond) {
		t.Fatalf("expected third ban ttl 24h, got %dms", ban3)
	}
}
