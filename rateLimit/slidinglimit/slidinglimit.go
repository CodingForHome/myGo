package slidinglimit

import (
	"context"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const slidingScript = `local limit = tonumber(ARGV[2])
local window = tonumber(ARGV[3])
local n = tonumber(ARGV[1])
local current = redis.call("INCRBY", KEYS[1], 1)
if current == 1 then
    redis.call("expire", KEYS[1], window)
end
for i=2,n do
	local temp = tonumber(redis.call("get", KEYS[i]))
	if temp > limit / n then
		return 0
	end
	current = current + temp 
end
if current < limit then
    return 1
elseif current == limit then
    return 2
else
    return 0
end`

type slidingLimit struct {
	period    int // 时间窗口的大小
	window    int // 小的时间窗口的大小
	quota     int // 限流阈值
	redis     *redis.Redis
	keyPrefix string // key的前缀
}

func NewSlidingLimit(period int, window int, quota int, redis *redis.Redis, keyPrefix string) *slidingLimit {
	return &slidingLimit{
		period:    period,
		window:    window,
		quota:     quota,
		redis:     redis,
		keyPrefix: keyPrefix,
	}
}

func (l slidingLimit) TakeCtx(ctx context.Context, now time.Time, key string) (int, error) {
	n := l.period / l.window // 这里大概表示一下，会有精度差
	keys := []string{}
	for i := 0; i < n; i++ {
		keys = append(keys, l.keyPrefix+key+strconv.FormatInt(now.Unix()-int64(i*l.window), 10))
	}
	_, err := l.redis.EvalCtx(ctx, slidingScript, keys,
		strconv.Itoa(n),
		strconv.Itoa(l.period),
		strconv.Itoa(l.quota))
	if err != nil {
		return 0, err
	}
	// 具体处理参考别处
	return 0, nil
}
