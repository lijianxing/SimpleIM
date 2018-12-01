package main

import (
	"flag"
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

	// router rpc
	if err := InitRouter(Conf.RouterRPCAddrs); err != nil {
		panic(err)
	}

	// start monitor
	if Conf.MonitorOpen {
		InitMonitor(Conf.MonitorAddrs)
	}

	// logic rpc
	if err := InitRPC(NewDefaultAuther()); err != nil {
		panic(err)
	}

	if err := InitHTTP(); err != nil {
		panic(err)
	}

	// block until a signal is received.
	InitSignal()
}
