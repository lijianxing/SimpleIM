package util

import (
	"time"

	"github.com/garyburd/redigo/redis"
)

type RedisConfig struct {
	Addr        string
	MaxIdle     int
	MaxActive   int
	IdleTimeout time.Duration
}

type RedisConnPool struct {
	pool *redis.Pool
	conf *RedisConfig
}

func (rp *RedisConnPool) Init(conf *RedisConfig) (err error) {
	rp.conf = conf
	rp.pool = &redis.Pool{
		MaxIdle:     conf.MaxIdle,
		MaxActive:   conf.MaxActive,
		IdleTimeout: conf.IdleTimeout,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", conf.Addr)
		},
	}
	return
}

func (rp *RedisConnPool) GetConn() redis.Conn {
	return rp.pool.Get()
}
