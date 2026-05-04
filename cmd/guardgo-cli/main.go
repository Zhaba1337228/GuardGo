package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	guardgo "github.com/Zhaba1337228/GuardGo"
)

type repEntry struct {
	Field string
	Score float64
}

type blockEntry struct {
	Key string
	TTL time.Duration
}

func main() {
	var (
		redisAddr   = flag.String("redis-addr", "127.0.0.1:6379", "Redis address")
		prefix      = flag.String("prefix", "github.com/Zhaba1337228/GuardGo", "GuardGo prefix")
		interval    = flag.Duration("interval", 2*time.Second, "Refresh interval")
		topN        = flag.Int("top", 10, "Top risk entities count")
		rulesetPath = flag.String("ruleset", "", "Optional ruleset path for DFA stats")
	)
	flag.Parse()

	rdb := redis.NewClient(&redis.Options{Addr: *redisAddr})
	ctx := context.Background()

	for {
		if err := drawDashboard(ctx, rdb, *prefix, *topN, *rulesetPath); err != nil {
			fmt.Fprintf(os.Stderr, "github.com/Zhaba1337228/GuardGo-cli: %v\n", err)
		}
		time.Sleep(*interval)
	}
}

func drawDashboard(ctx context.Context, rdb *redis.Client, prefix string, topN int, rulesetPath string) error {
	clearScreen()
	fmt.Printf("GuardGo CLI Dashboard | %s\n\n", time.Now().Format(time.RFC3339))

	entries, err := topRiskEntries(ctx, rdb, prefix+":rep", topN)
	if err != nil {
		return err
	}
	fmt.Printf("Top %d Reputation Scores\n", topN)
	if len(entries) == 0 {
		fmt.Println("  (no entries)")
	} else {
		for idx, entry := range entries {
			fmt.Printf("  %2d. %-50s %.2f\n", idx+1, entry.Field, entry.Score)
		}
	}

	blocks, err := activeBlocks(ctx, rdb, prefix+":bl:*", topN)
	if err != nil {
		return err
	}
	fmt.Println("\nActive Blacklist Keys")
	if len(blocks) == 0 {
		fmt.Println("  (none)")
	} else {
		for idx, block := range blocks {
			fmt.Printf("  %2d. %-40s ttl=%s\n", idx+1, block.Key, block.TTL.Truncate(time.Second))
		}
	}

	if strings.TrimSpace(rulesetPath) != "" {
		fmt.Println("\nDFA Ruleset Stats")
		rs, loadErr := guardgo.LoadRulesetFile(rulesetPath)
		if loadErr != nil {
			fmt.Printf("  load error: %v\n", loadErr)
		} else {
			stats := rs.Stats()
			fmt.Printf("  total_rules=%d total_nodes=%d\n", stats.TotalRules, stats.TotalNodes)
			fmt.Printf("  query_rules=%d header_rules=%d path_rules=%d any_rules=%d\n",
				stats.QueryRules, stats.HeaderRules, stats.PathRules, stats.AnyRules)
		}
	}

	fmt.Println("\nPress Ctrl+C to exit.")
	return nil
}

func topRiskEntries(ctx context.Context, rdb *redis.Client, hashKey string, topN int) ([]repEntry, error) {
	values, err := rdb.HGetAll(ctx, hashKey).Result()
	if err != nil {
		return nil, err
	}
	entries := make([]repEntry, 0, len(values))
	for field, raw := range values {
		score, convErr := strconv.ParseFloat(raw, 64)
		if convErr != nil {
			continue
		}
		entries = append(entries, repEntry{Field: field, Score: score})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Score > entries[j].Score })
	if len(entries) > topN {
		entries = entries[:topN]
	}
	return entries, nil
}

func activeBlocks(ctx context.Context, rdb *redis.Client, pattern string, topN int) ([]blockEntry, error) {
	keys, err := scanKeys(ctx, rdb, pattern)
	if err != nil {
		return nil, err
	}
	entries := make([]blockEntry, 0, len(keys))
	for _, key := range keys {
		ttl, ttlErr := rdb.PTTL(ctx, key).Result()
		if ttlErr != nil {
			continue
		}
		entries = append(entries, blockEntry{Key: key, TTL: ttl})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].TTL > entries[j].TTL })
	if len(entries) > topN {
		entries = entries[:topN]
	}
	return entries, nil
}

func scanKeys(ctx context.Context, rdb *redis.Client, pattern string) ([]string, error) {
	var (
		cursor uint64
		keys   []string
	)
	for {
		batch, next, err := rdb.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			return keys, nil
		}
	}
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}
