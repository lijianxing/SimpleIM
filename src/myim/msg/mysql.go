package main

import "myim/libs/util"

var (
	DBManager util.MysqlConnPool
)

func InitDBManager(config *util.DbConfig) (err error) {
	return DBManager.Init(config)
}
