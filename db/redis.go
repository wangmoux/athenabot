package db

import (
	"athenabot/config"
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

var RDB *redis.Client

func init() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     config.Conf.RedisHost,
		PoolSize: 10,
	})
	if ok, err := RDB.Ping(context.Background()).Result(); ok != "PONG" && err != nil {
		logrus.Panicln(ok, err)
	}

}
