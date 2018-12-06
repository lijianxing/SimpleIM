package main

import (
	inet "myim/libs/net"
	"myim/libs/net/xrpc"
	"myim/libs/proto"

	log "github.com/thinkboy/log4go"
)

var (
	msgRpcClient *xrpc.Clients
	msgRpcQuit   = make(chan struct{}, 1)

	msgService           = "MsgRPC"
	msgServicePing       = "MsgRPC.Ping"
	msgServiceSaveMsg    = "MsgRPC.SaveMsg"
	msgServiceGetMsgList = "MsgRPC.GetMsgList"
)

func InitMsgRpc(addrs []string) (err error) {
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
	msgRpcClient = xrpc.Dials(rpcOptions)
	// ping & reconnect
	msgRpcClient.Ping(msgServicePing)
	log.Info("init msg rpc: %v", rpcOptions)
	return
}

func saveMsg(arg *proto.SaveMsgArg) (reply *proto.SaveMsgReply, err error) {
	reply = &proto.SaveMsgReply{}
	if err = msgRpcClient.Call(msgServiceSaveMsg, &arg, &reply); err != nil {
		log.Error("c.Call(\"%s\", \"%v\", &ret) error(%v)", msgServiceSaveMsg, arg, err)
		return
	}
	return
}

func getMsgList(arg *proto.GetMsgListArg) (reply *proto.GetMsgListReply, err error) {
	reply = &proto.GetMsgListReply{}
	if err = msgRpcClient.Call(msgServiceGetMsgList, &arg, &reply); err != nil {
		log.Error("c.Call(\"%s\", \"%v\", &ret) error(%v)", msgServiceGetMsgList, arg, err)
		return
	}
	return
}
