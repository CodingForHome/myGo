package bucketlimit

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const bucketScript = `-- 每秒处理请求的速度
local rate = tonumber(ARGV[1])
-- 桶容量
local capacity = tonumber(ARGV[2])
-- 当前时间戳
local now = tonumber(ARGV[3])
-- 当前进水数量
local requested = tonumber(ARGV[4])
-- 需要多少秒才能把桶里面的水流完
local fill_time = capacity/rate
-- 向下取整,ttl为填满时间的2倍
local ttl = math.floor(fill_time*2)
-- 当前时间桶容量
local last_tokens = tonumber(redis.call("get", KEYS[1]))
-- 如果当前桶容量为0,说明是第一次进入,则默认容量为桶的最大容量
if last_tokens == nil then
last_tokens = capacity
end
-- 上一次刷新的时间
local last_refreshed = tonumber(redis.call("get", KEYS[2]))
-- 第一次进入则设置刷新时间为0
if last_refreshed == nil then
last_refreshed = 0
end
-- 距离上次请求的时间跨度
local delta = math.max(0, now-last_refreshed)
-- 距离上次请求的时间跨度,总共处理热多少请求,如果超多最大容量则丢弃多余的token
local filled_tokens = math.min(capacity, last_tokens+(delta*rate))
-- 本次请求token数量是否足够
local allowed = filled_tokens >= requested
-- 桶剩余数量
local new_tokens = filled_tokens
-- 允许本次token申请,计算剩余数量
if allowed then
new_tokens = filled_tokens - requested
end
-- 设置剩余token数量
redis.call("setex", KEYS[1], ttl, new_tokens)
-- 设置刷新时间
redis.call("setex", KEYS[2], ttl, now)

return allowed
`

type myBucketLimit struct {
	bucket    int          // 桶的容量
	rate      int          // 允许每秒请求的速率
	water     int          // 每秒加水的速率
	redis     *redis.Redis // redis服务端
	prefixKey string       // key的前缀
}

func NewMyBucketLimit(bucket, rate, water int, redis *redis.Redis, prefixKey string) *myBucketLimit {
	return &myBucketLimit{bucket, rate, water, redis, prefixKey}
}

func (l *myBucketLimit) Take(now time.Time, key string) (bool, error) {
	return l.TakeCtx(context.Background(), now, key)
}

func (l *myBucketLimit) TakeCtx(ctx context.Context, now time.Time, key string) (bool, error) {
	resp, err := l.redis.EvalCtx(ctx, bucketScript, []string{l.prefixKey + key, "timeStamp" + key},
		strconv.Itoa(l.rate),
		strconv.Itoa(l.bucket),
		strconv.FormatInt(now.Unix(), 10),
		strconv.Itoa(l.water))
	if err != nil {
		return false, err
	}
	res, ok := resp.(int64)
	if !ok {
		return false, errors.New("can't underStandThisResp")
	}
	return res == 1, nil
}
