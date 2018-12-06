package main

import "myim/libs/util"

var (
	RedisManager util.RedisConnPool
)

func InitRedis(conf *util.RedisConfig) (err error) {
	return RedisManager.Init(conf)
}
