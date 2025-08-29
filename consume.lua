local key = KEYS[1]
local limit = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])

local current = redis.call('GET', key)
if current == false then
    current = 0
else
    current = tonumber(current)
end

if current >= limit then
    return {current, limit, 0}
end

local new_value = redis.call('INCR', key)
if redis.call('TTL', key) == -1 then
    redis.call('EXPIRE', key, ttl)
end

if new_value > limit then
    redis.call('DECR', key)
    return {limit, limit, 0}
end

return {new_value, limit, 1}