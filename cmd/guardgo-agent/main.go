package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	var (
		redisAddr = flag.String("redis-addr", "127.0.0.1:6379", "Redis address")
		prefix    = flag.String("prefix", "github.com/Zhaba1337228/GuardGo", "GuardGo Redis key prefix")
		listen    = flag.String("listen", ":9090", "HTTP listen address")
		interval  = flag.Duration("interval", 15*time.Second, "Refresh interval")
	)
	flag.Parse()

	rdb := redis.NewClient(&redis.Options{Addr: *redisAddr})
	collector := newCollector(*prefix)
	prometheus.MustRegister(collector)

	go func() {
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()
		for {
			if err := collector.Refresh(context.Background(), rdb); err != nil {
				log.Printf("github.com/Zhaba1337228/GuardGo-agent refresh failed: %v", err)
			}
			<-ticker.C
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Printf("github.com/Zhaba1337228/GuardGo-agent serving metrics on %s/metrics", *listen)
	if err := http.ListenAndServe(*listen, nil); err != nil {
		log.Fatalf("github.com/Zhaba1337228/GuardGo-agent server error: %v", err)
	}
}

type collector struct {
	prefix string

	redisUp            prometheus.Gauge
	blacklistKeys      prometheus.Gauge
	rateLimitKeys      prometheus.Gauge
	threatKeys         prometheus.Gauge
	blacklistedMembers prometheus.Gauge
}

func newCollector(prefix string) *collector {
	return &collector{
		prefix: prefix,
		redisUp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "github.com/Zhaba1337228/GuardGo_agent_redis_up",
			Help: "Redis connectivity status (1 up, 0 down)",
		}),
		blacklistKeys: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "github.com/Zhaba1337228/GuardGo_agent_blacklist_keys_total",
			Help: "Total number of blacklist keys",
		}),
		rateLimitKeys: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "github.com/Zhaba1337228/GuardGo_agent_rate_limit_keys_total",
			Help: "Total number of rate-limit keys",
		}),
		threatKeys: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "github.com/Zhaba1337228/GuardGo_agent_threat_keys_total",
			Help: "Total number of threat keys",
		}),
		blacklistedMembers: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "github.com/Zhaba1337228/GuardGo_agent_blacklisted_members_total",
			Help: "Total number of members across blacklist keys",
		}),
	}
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	c.redisUp.Describe(ch)
	c.blacklistKeys.Describe(ch)
	c.rateLimitKeys.Describe(ch)
	c.threatKeys.Describe(ch)
	c.blacklistedMembers.Describe(ch)
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.redisUp.Collect(ch)
	c.blacklistKeys.Collect(ch)
	c.rateLimitKeys.Collect(ch)
	c.threatKeys.Collect(ch)
	c.blacklistedMembers.Collect(ch)
}

func (c *collector) Refresh(ctx context.Context, rdb *redis.Client) error {
	if err := rdb.Ping(ctx).Err(); err != nil {
		c.redisUp.Set(0)
		return err
	}
	c.redisUp.Set(1)

	blacklistKeys, err := scanCount(ctx, rdb, c.prefix+":bl:*")
	if err != nil {
		return err
	}
	rateLimitKeys, err := scanCount(ctx, rdb, c.prefix+":rl:*")
	if err != nil {
		return err
	}
	threatKeys, err := scanCount(ctx, rdb, c.prefix+":threat:*")
	if err != nil {
		return err
	}
	blacklistedMembers, err := sumSetMembers(ctx, rdb, c.prefix+":bl:*")
	if err != nil {
		return err
	}

	c.blacklistKeys.Set(float64(blacklistKeys))
	c.rateLimitKeys.Set(float64(rateLimitKeys))
	c.threatKeys.Set(float64(threatKeys))
	c.blacklistedMembers.Set(float64(blacklistedMembers))
	return nil
}

func scanCount(ctx context.Context, rdb *redis.Client, pattern string) (int64, error) {
	var (
		cursor uint64
		total  int64
	)
	for {
		keys, next, err := rdb.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return 0, err
		}
		total += int64(len(keys))
		cursor = next
		if cursor == 0 {
			return total, nil
		}
	}
}

func sumSetMembers(ctx context.Context, rdb *redis.Client, pattern string) (int64, error) {
	var (
		cursor uint64
		total  int64
	)
	for {
		keys, next, err := rdb.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return 0, err
		}
		for _, key := range keys {
			cnt, err := rdb.SCard(ctx, key).Result()
			if err != nil {
				return 0, err
			}
			total += cnt
		}
		cursor = next
		if cursor == 0 {
			return total, nil
		}
	}
}
