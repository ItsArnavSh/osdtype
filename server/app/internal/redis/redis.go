package redis

import (
	"github.com/go-redis/redis"
	"github.com/spf13/viper"
)

type RedisClient struct {
	redis_conn *redis.Client
}

func NewRedisClient() RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     viper.GetString("Redis.addr"),
		Password: "",
		DB:       0,
	})
	return RedisClient{rdb}
}
