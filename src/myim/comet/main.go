package main

import (
	"flag"
	"runtime"

	log "github.com/thinkboy/log4go"
)

var (
	DefaultServer *Server
	Debug         bool
)

func main() {
	flag.Parse()
	if err := InitConfig(); err != nil {
		panic(err)
	}

	Debug = Conf.Debug
	runtime.GOMAXPROCS(Conf.MaxProc)
	log.LoadConfiguration(Conf.Log)
	defer log.Close()

	log.Info("comet[%s] start", Ver)

	// logic rpc
	if err := InitLogicRpc(Conf.LogicAddrs); err != nil {
		panic(err)
	}

	// start monitor
	if Conf.MonitorOpen {
		InitMonitor(Conf.MonitorAddrs)
	}

	// new stat
	stat := NewStat()

	// new server
	buckets := make([]*Bucket, Conf.Bucket)
	for i := 0; i < Conf.Bucket; i++ {
		buckets[i] = NewBucket(BucketOptions{
			ChannelSize: Conf.BucketChannel,
		})
	}

	// 缓冲区/定时器资源 (Round只是做一个锁分段的作用, Round数组的元素自身必须是线程安全的)
	round := NewRound(RoundOptions{
		Reader:       Conf.TCPReader,
		ReadBuf:      Conf.TCPReadBuf,
		ReadBufSize:  Conf.TCPReadBufSize,
		Writer:       Conf.TCPWriter,
		WriteBuf:     Conf.TCPWriteBuf,
		WriteBufSize: Conf.TCPWriteBufSize,
		Timer:        Conf.Timer,
		TimerSize:    Conf.TimerSize,
	})

	operator := new(DefaultOperator)
	DefaultServer = NewServer(stat, buckets, round, operator, ServerOptions{
		CliProto:         Conf.CliProto,
		SvrProto:         Conf.SvrProto,
		HandshakeTimeout: Conf.HandshakeTimeout,
		TCPKeepalive:     Conf.TCPKeepalive,
		TCPRcvbuf:        Conf.TCPRcvbuf,
		TCPSndbuf:        Conf.TCPSndbuf,
	})

	// tcp comet
	if err := InitTCP(Conf.TCPBind, Conf.MaxProc); err != nil {
		panic(err)
	}

	// websocket comet
	if err := InitWebsocket(Conf.WebsocketBind, Conf.MaxProc); err != nil {
		panic(err)
	}

	// start rpc
	if err := InitRPCPush(Conf.RPCPushAddrs); err != nil {
		panic(err)
	}

	// block until a signal is received.
	InitSignal()
}
