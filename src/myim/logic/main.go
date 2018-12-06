package main

import (
	"flag"
	"myim/libs/util"
	"runtime"

	log "github.com/thinkboy/log4go"
)

var (
	DefaultStat *Stat
)

func main() {
	flag.Parse()

	if err := InitConfig(); err != nil {
		panic(err)
	}
	runtime.GOMAXPROCS(Conf.MaxProc)
	log.LoadConfiguration(Conf.Log)
	defer log.Close()
	log.Info("logic[%s] start", Ver)

	DefaultStat = NewStat()

	// init mysql
	if err := InitDBManager(&util.DbConfig{
		Dsn:     Conf.DbDsn,
		MaxOpen: Conf.DbMaxOpen,
		MaxIdle: Conf.DbMaxIdle,
	}); err != nil {
		panic(err)
	}

	// init redis
	if err := InitRedis(&util.RedisConfig{
		Addr: Conf.RedisAddr, MaxActive: Conf.RedisPoolMaxActive, MaxIdle: Conf.RedisPoolMaxIdle,
		IdleTimeout: Conf.RedisPoolIdleTimeout,
	}); err != nil {
		panic(err)
	}

	// init comet
	if err := InitComet(Conf.CometRPCAddrs, CometOptions{
		RoutineSize: Conf.RoutineSize,
		RoutineChan: Conf.RoutineChan,
	}); err != nil {
		panic(err)
	}

	// msg rpc
	if err := InitMsgRpc(Conf.MsgAddrs); err != nil {
		panic(err)
	}

	// logic rpc
	if err := InitRPC(NewDefaultAuther()); err != nil {
		panic(err)
	}

	if err := InitHTTP(); err != nil {
		panic(err)
	}

	// start monitor
	if Conf.MonitorOpen {
		InitMonitor(Conf.MonitorAddrs)
	}

	// block until a signal is received.
	InitSignal()
}
