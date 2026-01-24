package redis

import (
	"context"
	"time"
	"github.com/gospacex/gospacex/core/storage/conf"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

var (
	RC *redis.Client
)

func Init(chainEnable bool, cfg *conf.RedisConfig) (err error) {
	ctx := context.Background()
	readTimeout := time.Duration(30) * time.Second
	writeTimeout := time.Duration(30) * time.Second
	RC = redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		DB:           cfg.Db,
		Password:     cfg.Password,
		PoolSize:     cfg.PoolSize,
		MaxIdleConns: cfg.MaxIdle,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	})
	// 链路追踪
	if chainEnable == true {
		// 启用 tracing
		if err := redisotel.InstrumentTracing(RC); err != nil {
			panic(err)
		}
		// 启用 metrics
		if err := redisotel.InstrumentMetrics(RC); err != nil {
			panic(err)
		}
	}
	if _, err = RC.Ping(ctx).Result(); err != nil {
		return err
	}
	return nil
}
