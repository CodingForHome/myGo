package periodlimit

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

// to be compatible with aliyun redis, we cannot use `local key = KEYS[1]` to reuse the key
const periodScript = `local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local current = redis.call("INCRBY", KEYS[1], 1)
if current == 1 then
    redis.call("expire", KEYS[1], window)
end
if current < limit then
    return 1
elseif current == limit then
    return 2
else
    return 0
end`

const (
	// Unknown means not initialized state.
	Unknown = iota
	// Allowed means allowed state.
	Allowed
	// HitQuota means this request exactly hit the quota.
	HitQuota
	// OverQuota means passed the quota.
	OverQuota

	internalOverQuota = 0
	internalAllowed   = 1
	internalHitQuota  = 2
)

// ErrUnknownCode is an error that represents unknown status code.
var ErrUnknownCode = errors.New("unknown status code")

type (
	// PeriodOption defines the method to customize a PeriodLimit.
	// 设置一个别名
	PeriodOption func(l *PeriodLimit)

	// A PeriodLimit is used to limit requests during a period of time.
	PeriodLimit struct {
		period     int // 时间窗口的大小
		quota      int // 限流阈值
		limitStore *redis.Redis
		keyPrefix  string // key的前缀
		align      bool   // false时period固定，为true时period具有周期性：3-2-1-3-2-1...
	}
)

// NewPeriodLimit returns a PeriodLimit with given parameters.
// 初始化一个限流器
func NewPeriodLimit(period, quota int, limitStore *redis.Redis, keyPrefix string,
	opts ...PeriodOption) *PeriodLimit {
	limiter := &PeriodLimit{
		period:     period,
		quota:      quota,
		limitStore: limitStore,
		keyPrefix:  keyPrefix,
	}

	for _, opt := range opts {
		opt(limiter)
	}

	return limiter
}

// Take requests a permit, it returns the permit state.
func (h *PeriodLimit) Take(key string) (int, error) {
	return h.TakeCtx(context.Background(), key)
}

// TakeCtx requests a permit with context, it returns the permit state.
// 返回key的限流结果
func (h *PeriodLimit) TakeCtx(ctx context.Context, key string) (int, error) {
	// 执行lua脚本进行限流判断
	resp, err := h.limitStore.EvalCtx(ctx, periodScript, []string{h.keyPrefix + key}, []string{
		strconv.Itoa(h.quota),               // 限流的阈值
		strconv.Itoa(h.calcExpireSeconds()), // 窗口的大小
	})
	if err != nil {
		return Unknown, err
	}
	// 判断限流结果
	code, ok := resp.(int64)
	if !ok {
		return Unknown, ErrUnknownCode
	}

	switch code {
	case internalOverQuota: // 超过阈值
		return OverQuota, nil
	case internalAllowed: // 小于阈值
		return Allowed, nil
	case internalHitQuota: // 等于阈值
		return HitQuota, nil
	default:
		return Unknown, ErrUnknownCode
	}
}

// 这段代码用于计算一个周期内的剩余秒数。如果h.align为true，则会计算当前时间到下一个周期的剩余秒数，否则返回h.period的值。
func (h *PeriodLimit) calcExpireSeconds() int {
	if h.align {
		now := time.Now()
		_, offset := now.Zone()
		unix := now.Unix() + int64(offset)
		return h.period - int(unix%int64(h.period))
	}

	return h.period
}

// Align returns a func to customize a PeriodLimit with alignment.
// For example, if we want to limit end users with 5 sms verification messages every day,
// we need to align with the local timezone and the start of the day.
func Align() PeriodOption {
	return func(l *PeriodLimit) {
		l.align = true
	}
}
