package periodlimit

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type myPeriodLimit struct {
	period    int // 时间周期
	quota     int // 阈值
	redis     *redis.Redis
	keyPrefix string // 前缀
}

func NewMyPeriodLimit(period, quota int, redis *redis.Redis, keyPrefix string) *myPeriodLimit {
	return &myPeriodLimit{
		period:    period,
		quota:     quota,
		redis:     redis,
		keyPrefix: keyPrefix,
	}
}
func (l myPeriodLimit) Take(key string) (int, error) {
	return l.TakeCtx(context.Background(), key)
}
func (l myPeriodLimit) TakeCtx(ctx context.Context, key string) (int, error) {
	resp, err := l.redis.EvalCtx(ctx, periodScript, []string{l.keyPrefix + key}, strconv.Itoa(l.quota), strconv.Itoa(l.period))
	if err != nil {
		return Unknown, err
	}

	code, ok := resp.(int64)
	if !ok {
		return Unknown, ErrUnknownCode
	}
	switch code {
	case internalOverQuota:
		return OverQuota, nil
	case internalAllowed:
		return Allowed, nil
	case internalHitQuota:
		return HitQuota, nil
	default:
		return Unknown, ErrUnknownCode
	}
}
