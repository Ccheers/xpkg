package core

const pendingRecordPrelude = `
local function unpackPending(value)
    local ok, record = pcall(cmsgpack.unpack, value)
    if not ok or type(record) ~= 'table' or #record ~= 5 then
        error('invalid pending record')
    end
    for key, field in pairs(record) do
        if type(key) ~= 'number' or key < 1 or key > 5 or key % 1 ~= 0 or type(field) ~= 'string' then
            error('invalid pending record')
        end
    end
    for index = 1, 5 do
        if type(record[index]) ~= 'string' or (index < 5 and record[index] == '') then
            error('invalid pending record')
        end
    end
    return record
end
`

const claimScript = luaPrelude + pendingRecordPrelude + `
if #KEYS ~= 4 or #ARGV ~= 3 or ARGV[1] == '' or ARGV[2] == '' or ARGV[3] == '' then
    error('invalid claim arguments')
end
checkType(KEYS[1], 'hash')
checkType(KEYS[2], 'zset')
checkType(KEYS[3], 'hash')
checkType(KEYS[4], 'list')
local generation = redis.call('HGET', KEYS[3], ARGV[1])
if generation ~= ARGV[3] then
    return {'channel', 'changed'}
end
local pending = redis.call('HGET', KEYS[1], ARGV[2])
local due = redis.call('ZSCORE', KEYS[2], ARGV[2])
if pending then
    local record = unpackPending(pending)
    if record[1] ~= KEYS[3] or record[2] ~= ARGV[1] or record[3] ~= generation or record[4] ~= KEYS[4] then
        error('receipt belongs to another channel')
    end
    if not due then
        error('pending receipt has no due index')
    end
    return {record[5]}
end
if due then
    error('pending due index has no record')
end
local data = redis.call('LINDEX', KEYS[4], 0)
if not data then
    return {}
end
local record = cmsgpack.pack({KEYS[3], ARGV[1], generation, KEYS[4], data})
local dueAt = nowMillis() + 120000
-- 先保存可恢复凭据，再移除队首；类型、元组编码及时间计算均已在写入前完成。
redis.call('HSET', KEYS[1], ARGV[2], record)
redis.call('ZADD', KEYS[2], dueAt, ARGV[2])
redis.call('LPOP', KEYS[4])
return {data}
`

const acknowledgeScript = luaPrelude + `
if #KEYS ~= 2 or #ARGV ~= 1 or ARGV[1] == '' then
    error('invalid acknowledge arguments')
end
checkType(KEYS[1], 'hash')
checkType(KEYS[2], 'zset')
redis.call('HDEL', KEYS[1], ARGV[1])
redis.call('ZREM', KEYS[2], ARGV[1])
return 1
`

const pendingCandidatesScript = luaPrelude + `
local limit = tonumber(ARGV[1])
if #KEYS ~= 2 or #ARGV ~= 1 or not limit or limit < 1 or limit > 1000 or limit % 1 ~= 0 then
    error('invalid recovery limit')
end
checkType(KEYS[1], 'hash')
checkType(KEYS[2], 'zset')
return redis.call('ZRANGEBYSCORE', KEYS[2], '-inf', nowMillis(), 'LIMIT', 0, limit)
`

const deferRecoveryScript = luaPrelude + `
checkType(KEYS[1], 'zset')
local due = redis.call('ZSCORE', KEYS[1], ARGV[1])
local now = nowMillis()
if due and tonumber(due) <= now then
    redis.call('ZADD', KEYS[1], 'XX', now + 300000, ARGV[1])
end
return 1
`

const recoverScript = luaPrelude + pendingRecordPrelude + `
if #KEYS ~= 2 or #ARGV ~= 1 or ARGV[1] == '' then
    error('invalid recover arguments')
end
checkType(KEYS[1], 'hash')
checkType(KEYS[2], 'zset')
local due = redis.call('ZSCORE', KEYS[2], ARGV[1])
local pending = redis.call('HGET', KEYS[1], ARGV[1])
if not due and not pending then
    return 0
end
if not due or not pending then
    error('inconsistent pending receipt')
end
if tonumber(due) > nowMillis() then
    return 0
end
local record = unpackPending(pending)
checkType(record[1], 'hash')
checkType(record[4], 'list')
local generation = redis.call('HGET', record[1], record[2])
if generation ~= record[3] then
    redis.call('HDEL', KEYS[1], ARGV[1])
    redis.call('ZREM', KEYS[2], ARGV[1])
    return 0
end
redis.call('RPUSH', record[4], record[5])
redis.call('EXPIRE', record[4], 600)
redis.call('HDEL', KEYS[1], ARGV[1])
redis.call('ZREM', KEYS[2], ARGV[1])
return 1
`
