package main

import (
	inet "myim/libs/net"
	"myim/libs/net/xrpc"
	"myim/libs/proto"
	"time"

	log "github.com/thinkboy/log4go"
)

var (
	logicRpcClient *xrpc.Clients
	logicRpcQuit   = make(chan struct{}, 1)

	logicService           = "RPC"
	logicServicePing       = "RPC.Ping"
	logicServiceConnect    = "RPC.Connect"
	logicServiceDisconnect = "RPC.Disconnect"
	logicServiceOperate    = "RPC.Operate"
)

func InitLogicRpc(addrs []string) (err error) {
	var (
		bind          string
		network, addr string
		rpcOptions    []xrpc.ClientOptions
	)
	for _, bind = range addrs {
		if network, addr, err = inet.ParseNetwork(bind); err != nil {
			log.Error("inet.ParseNetwork() error(%v)", err)
			return
		}
		options := xrpc.ClientOptions{
			Proto: network,
			Addr:  addr,
		}
		rpcOptions = append(rpcOptions, options)
	}
	// rpc clients
	logicRpcClient = xrpc.Dials(rpcOptions)
	// ping & reconnect
	logicRpcClient.Ping(logicServicePing)
	log.Info("init logic rpc: %v", rpcOptions)
	return
}

func connect(p *proto.Proto) (key string, heartbeat time.Duration, err error) {
	var (
		arg   = proto.ConnArg{Server: Conf.ServerId, Data: p.Body}
		reply = proto.ConnReply{}
	)
	if err = logicRpcClient.Call(logicServiceConnect, &arg, &reply); err != nil {
		log.Error("c.Call(\"%s\", \"%v\", &ret) error(%v)", logicServiceConnect, arg, err)
		return
	}
	if !reply.Ok {
		log.Error("auth failed.loginReq:%s", string(p.Body))
		err = ErrAuthFailed
		return
	}

	key = reply.Key
	heartbeat = 60 * time.Second
	return
}

func disconnect(key string) (has bool, err error) {
	var (
		arg   = proto.DisconnArg{Server: Conf.ServerId, Key: key}
		reply = proto.DisconnReply{}
	)
	if err = logicRpcClient.Call(logicServiceDisconnect, &arg, &reply); err != nil {
		log.Error("c.Call(\"%s\", \"%v\", &ret) error(%v)", logicServiceDisconnect, arg, err)
		return
	}
	has = reply.Has
	return
}

func operate(key string, p *proto.Proto) (err error) {
	var (
		arg   = proto.OperArg{Server: Conf.ServerId, Key: key, SeqId: p.SeqId, Op: p.Operation, Data: p.Body}
		reply = proto.OperReply{}
	)
	if err = logicRpcClient.Call(logicServiceOperate, &arg, &reply); err != nil {
		log.Error("c.Call(\"%s\", \"%v\", &ret) error(%v)", logicServiceOperate, arg, err)
		return
	}

	p.Operation = reply.Op
	p.Body = reply.Data

	return
}
