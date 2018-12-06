package main

import (
	"flag"
	"myim/libs/util"
	"runtime"

	log "github.com/thinkboy/log4go"
)

const (
	VERSION = "0.1"
)

func main() {
	flag.Parse()
	if err := InitConfig(); err != nil {
		panic(err)
	}
	runtime.GOMAXPROCS(Conf.MaxProc)
	log.LoadConfiguration(Conf.Log)
	defer log.Close()
	log.Info("msg[%s] start", VERSION)

	// init mysql
	if err := InitDBManager(&util.DbConfig{
		Dsn:     Conf.DbDsn,
		MaxOpen: Conf.DbMaxOpen,
		MaxIdle: Conf.DbMaxIdle,
	}); err != nil {
		panic(err)
	}

	// start rpc
	buckets := make([]*Bucket, Conf.Bucket)
	for i := 0; i < Conf.Bucket; i++ {
		buckets[i] = NewBucket(Conf.ChatMsgNum, Conf.Chat, Conf.ChatExpire)
	}
	if err := InitRPC(buckets); err != nil {
		panic(err)
	}

	// start monitor
	if Conf.MonitorOpen {
		InitMonitor(Conf.MonitorAddrs)
	}

	// block until a signal is received.
	InitSignal()
}
