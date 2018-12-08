package main

import (
	"fmt"
	"myim/libs/hash/cityhash"
	inet "myim/libs/net"
	"myim/libs/proto"
	"net"
	"net/rpc"

	log "github.com/thinkboy/log4go"
)

func InitRPC(bs []*Bucket) (err error) {
	var (
		network, addr string
		c             = &MsgRPC{Buckets: bs, BucketIdx: uint32(len(bs))}
	)
	rpc.Register(c)
	for i := 0; i < len(Conf.RPCAddrs); i++ {
		log.Info("start listen rpc addr: \"%s\"", Conf.RPCAddrs[i])
		if network, addr, err = inet.ParseNetwork(Conf.RPCAddrs[i]); err != nil {
			log.Error("inet.ParseNetwork() error(%v)", err)
			return
		}
		go rpcListen(network, addr)
	}
	return
}

func rpcListen(network, addr string) {
	l, err := net.Listen(network, addr)
	if err != nil {
		log.Error("net.Listen(\"%s\", \"%s\") error(%v)", network, addr, err)
		panic(err)
	}
	// if process exit, then close the rpc bind
	defer func() {
		log.Info("rpc addr: \"%s\" close", addr)
		if err := l.Close(); err != nil {
			log.Error("listener.Close() error(%v)", err)
		}
	}()
	rpc.Accept(l)
}

// Msg RPC
type MsgRPC struct {
	Buckets   []*Bucket
	BucketIdx uint32
}

func (r *MsgRPC) bucket(key string) *Bucket {
	idx := cityhash.CityHash32([]byte(key), uint32(len(key))) % r.BucketIdx
	// fix panic
	if idx < 0 {
		idx = 0
	}
	return r.Buckets[idx]
}

func (r *MsgRPC) Ping(arg *proto.NoArg, reply *proto.NoReply) error {
	return nil
}

func (r *MsgRPC) SaveMsg(arg *proto.SaveMsgArg, reply *proto.SaveMsgReply) (err error) {
	if arg == nil {
		log.Error("save msg arg is nil")
		err = ErrInvalidArgument
		return
	}
	log.Debug("receive savemsg req:%v", *arg)
	chatKey := genChatKey(arg.AppId, arg.ChatType, arg.ChatCode)
	if err = r.bucket(chatKey).SaveMsg(chatKey, arg, reply); err != nil {
		log.Error("save msg failed.arg:%v, err:%v", *arg, err)
	}
	return
}

func (r *MsgRPC) GetMsgList(arg *proto.GetMsgListArg, reply *proto.GetMsgListReply) (err error) {
	if arg == nil {
		log.Error("get msg list arg is nil")
		err = ErrInvalidArgument
		return
	}
	chatKey := genChatKey(arg.AppId, arg.ChatType, arg.ChatCode)
	if err = r.bucket(chatKey).GetMsgList(chatKey, arg, reply); err != nil {
		log.Error("get msg list failed.arg:%v, err:%v", *arg, err)
	}
	return
}

func genChatKey(appId string, chatType int32, chatCode string) string {
	return fmt.Sprintf("%s_%d_%s", appId, chatType, chatCode)
}
