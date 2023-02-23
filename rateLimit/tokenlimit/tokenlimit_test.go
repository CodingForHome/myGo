package tokenlimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	burst = 20 // 桶的容量
	rate  = 1  // 产生令牌的速率

)

func TestTokenLimit(t *testing.T) {
	myRedis := redis.RedisConf{Host: "localhost:6379", Pass: "123456"}.NewRedis()
	tokenLimit := NewMyTokenLimit(burst, rate, myRedis, "tokenLimit")
	take, err := tokenLimit.Take(time.Now(), "JinFeng", 10)
	if err != nil {
		return
	}
	assert.Equal(t, take, true)
}
