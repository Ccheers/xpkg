package core

const luaPrelude = `
local function checkType(key, expected)
    local actual = redis.call('TYPE', key).ok
    if actual ~= 'none' and actual ~= expected then
        error('WRONGTYPE ' .. key .. ' must be ' .. expected)
    end
end
local function nowMillis()
    local tm = redis.call('TIME')
    return tonumber(tm[1]) * 1000 + math.floor(tonumber(tm[2]) / 1000)
end
`

const registerScript = luaPrelude + `
checkType(KEYS[1], 'hash')
checkType(KEYS[2], 'zset')
checkType(KEYS[3], 'list')
local ttl = tonumber(ARGV[4])
if not ttl or ttl <= 0 then error('invalid subscription TTL') end
local generation = redis.call('HGET', KEYS[1], ARGV[1])
if not generation then
    redis.call('DEL', KEYS[2], KEYS[3])
    generation = ARGV[3]
end
redis.call('ZADD', KEYS[2], nowMillis() + ttl, ARGV[2])
redis.call('HSET', KEYS[1], ARGV[1], generation)
return 1
`

const unregisterScript = luaPrelude + `
checkType(KEYS[1], 'hash')
checkType(KEYS[2], 'zset')
checkType(KEYS[3], 'list')
redis.call('ZREM', KEYS[2], ARGV[2])
redis.call('ZREMRANGEBYSCORE', KEYS[2], '-inf', nowMillis())
if redis.call('ZCARD', KEYS[2]) == 0 then
    redis.call('HDEL', KEYS[1], ARGV[1])
    redis.call('DEL', KEYS[2], KEYS[3])
end
return 1
`

const sweepScript = luaPrelude + `
checkType(KEYS[1], 'hash')
checkType(KEYS[2], 'zset')
checkType(KEYS[3], 'list')
if redis.call('HGET', KEYS[1], ARGV[1]) ~= ARGV[2] then return 0 end
redis.call('ZREMRANGEBYSCORE', KEYS[2], '-inf', nowMillis())
if redis.call('ZCARD', KEYS[2]) ~= 0 then return 0 end
redis.call('HDEL', KEYS[1], ARGV[1])
redis.call('DEL', KEYS[2], KEYS[3])
return 1
`

const addChannelScript = luaPrelude + `
checkType(KEYS[1], 'hash')
checkType(KEYS[2], 'zset')
checkType(KEYS[3], 'list')
if redis.call('HEXISTS', KEYS[1], ARGV[1]) == 0 then
    redis.call('DEL', KEYS[2], KEYS[3])
    redis.call('HSET', KEYS[1], ARGV[1], ARGV[2])
end
return 1
`

const removeChannelScript = luaPrelude + `
checkType(KEYS[1], 'hash')
checkType(KEYS[2], 'zset')
checkType(KEYS[3], 'list')
redis.call('HDEL', KEYS[1], ARGV[1])
redis.call('DEL', KEYS[2], KEYS[3])
return 1
`

const channelsScript = luaPrelude + `
checkType(KEYS[1], 'hash')
return redis.call('HGETALL', KEYS[1])
`

const generationScript = luaPrelude + `
checkType(KEYS[1], 'hash')
return redis.call('HGET', KEYS[1], ARGV[1]) or ''
`

const pushScript = luaPrelude + `
checkType(KEYS[1], 'hash')
checkType(KEYS[2], 'list')
if redis.call('HGET', KEYS[1], ARGV[1]) ~= ARGV[2] then return 0 end
redis.call('RPUSH', KEYS[2], ARGV[3])
redis.call('EXPIRE', KEYS[2], 600)
return 1
`
