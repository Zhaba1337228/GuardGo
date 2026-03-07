package redislua

import "github.com/redis/go-redis/v9"

// ReputationScript updates reputation score and applies penalty/blacklist decision.
//
// KEYS:
// 1) reputation hash key
// 2) penalty key
// 3) blacklist set key
// 4) ban attempts key
//
// ARGV:
// 1) field
// 2) score_delta
// 3) score_window_ms
// 4) warning_level
// 5) threshold
// 6) penalty_ttl_ms
// 7) backoff_window_ms
// 8) ban_level_1_ms
// 9) ban_level_2_ms
// 10) ban_level_3_ms
//
// Returns: {score, action, ban_ttl, attempts}
// action: 0=none, 1=penalty, 2=blacklist
const reputationLua = `
local field = ARGV[1]
local score_delta = tonumber(ARGV[2])
local score_window = tonumber(ARGV[3])
local warning = tonumber(ARGV[4])
local threshold = tonumber(ARGV[5])
local penalty_ttl = tonumber(ARGV[6])
local backoff_window = tonumber(ARGV[7])
local ban1 = tonumber(ARGV[8])
local ban2 = tonumber(ARGV[9])
local ban3 = tonumber(ARGV[10])

local score = tonumber(redis.call("HINCRBYFLOAT", KEYS[1], field, score_delta))
redis.call("PEXPIRE", KEYS[1], score_window)

if score >= threshold then
  local attempts = redis.call("INCR", KEYS[4])
  if attempts == 1 then
    redis.call("PEXPIRE", KEYS[4], backoff_window)
  end
  local ban_ttl = ban1
  if attempts == 2 then
    ban_ttl = ban2
  elseif attempts >= 3 then
    ban_ttl = ban3
  end
  redis.call("SADD", KEYS[3], "1")
  redis.call("PEXPIRE", KEYS[3], ban_ttl)
  redis.call("DEL", KEYS[2])
  return {score, 2, ban_ttl, attempts}
end

if score >= warning then
  redis.call("SET", KEYS[2], "1", "PX", penalty_ttl)
  return {score, 1, 0, 0}
end

return {score, 0, 0, 0}
`

var ReputationScript = redis.NewScript(reputationLua)
