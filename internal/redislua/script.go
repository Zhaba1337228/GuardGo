package redislua

import "github.com/redis/go-redis/v9"

// criticalPathLua performs:
// 1) SISMEMBER on a per-IP blacklist SET key (member "1")
// 2) INCR on rate-limit counter key
// 3) PEXPIRE on first INCR (ttl setup)
//
// Returns:
// - 2 if blacklisted
// - 1 if rate limited
// - 0 if allowed
// plus current counter value and reset ttl in milliseconds.
const criticalPathLua = `
local is_bl = redis.call("SISMEMBER", KEYS[1], "1")
if is_bl == 1 then
  local bl_ttl = redis.call("PTTL", KEYS[1])
  return {2, 0, bl_ttl}
end

local cur = redis.call("INCR", KEYS[2])
if cur == 1 then
  redis.call("PEXPIRE", KEYS[2], ARGV[1])
end

local ttl = redis.call("PTTL", KEYS[2])
if cur > tonumber(ARGV[2]) then
  return {1, cur, ttl}
end

return {0, cur, ttl}
`

var CriticalPathScript = redis.NewScript(criticalPathLua)
