package main

import (
	"encoding/json"
	"myim/libs/define"
	inet "myim/libs/net"
	"myim/libs/proto"
	"net"
	"net/rpc"

	log "github.com/thinkboy/log4go"
)

func InitRPC(auther Auther) (err error) {
	var (
		network, addr string
		c             = &RPC{auther: auther}
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

// RPC
type RPC struct {
	auther Auther
}

func (r *RPC) Ping(arg *proto.NoArg, reply *proto.NoReply) error {
	return nil
}

// Connect auth and registe login
func (r *RPC) Connect(arg *proto.ConnArg, reply *proto.ConnReply) (err error) {
	if arg == nil {
		err = ErrConnectArgs
		log.Error("Connect() error(%v)", err)
		return
	}

	var loginReq LoginReq
	if err = json.Unmarshal(arg.Data, &loginReq); err != nil {
		return
	}

	if key, resp, err = doLogin(loginReq); err != nil {
		return
	}

	reply.Ok = true
	reply.Key = key

	return
}

// Disconnect notice router offline
func (r *RPC) Disconnect(arg *proto.DisconnArg, reply *proto.DisconnReply) (err error) {
	if arg == nil {
		err = ErrDisconnectArgs
		log.Error("Disconnect() error(%v)", err)
		return
	}

	var req LogoutReq
	if err = json.Unmarshal(arg.Data, &req); err != nil {
		return
	}

	if resp, err = doLogout(req); err != nil {
		return
	}

	reply.Ok = true

	return
}

// 消息处理
func (r *RPC) Operate(arg *proto.OperArg, reply *proto.OperReply) (err error) {
	if arg == nil {
		err = ErrOperateArgs
		log.Error("Operate() error(%v)", err)
		return
	}

	var (
		op   int32 = define.OP_NONE
		resp interface{}
	)
	switch arg.Op {
	case define.OP_HEARTBEAT:
		var hbReq HeartbeatReq
		if err = json.Unmarshal(arg.Data, &hbReq); err != nil {
			return
		}
		resp, err = doHeartbeat(arg.Key, hbReq)
		reply.Op = define.OP_HEARTBEAT_REPLY

	case define.OP_SEND_MSG:
		var msgReq SendMsgReq
		if err = json.Unmarshal(arg.Data, &msgReq); err != nil {
			return
		}
		resp, err = doSendMsg(arg.Key, msgReq)
		reply.Op = define.OP_SEND_MSG_REPLY

	case define.OP_OP_MSG_SYNC:
		// TODO
		reply.Op = define.OP_OP_MSG_SYNC_REPLY

	default:
		log.Error("Operate operation not found. op=%d", arg.Op)
		err = ErrUnknownOper
		return
	}

	if err != nil {
		log.Error("Operate process error(%v)", err)
		return
	}

	if op != define.OP_NONE && resp != nil {
		if reply.Data, err = json.Marshal(resp); err != nil {
			return
		}
	}
	reply.Op = op

	return
}

///////////////  logic /////////////

func doLogin(loginReq LoginReq) (key string, req LoginResp, err error) {
	if req == nil {
		err = ErrInvalidArgument
		return
	}

	var (
		appId  string
		userId string
	)
	if loginReq.UserInfo != nil {
		appId = loginReq.UserInfo.AppId
		userId = loginReq.UserInfo.UserId
	}

	if ok := r.auther.Auth(loginReq.UserInfo.AppId, loginReq.UserId, loginReq.Token); ok {
		// 设置router
		if err = connect(arg.Server); err != nil {
			return
		}
	} else {
		err = ErrAuthFailed
	}
	return
}

func doLogout(key string, logoutReq LogoutReq) (resp LogoutResp, err error) {
}

func doHeartbeat(key string, hbReq HeartbeatReq) (resp HeartbeatResp, err error) {
}

func doSendMsg(key string, msgReq SendMsgReq) (resp SendMsgResp, err error) {
}
