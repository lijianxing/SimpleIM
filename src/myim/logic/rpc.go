package main

import (
	"encoding/json"
	"myim/libs/define"
	inet "myim/libs/net"
	"myim/libs/proto"
	"myim/libs/util"
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
		log.Error("connect arg is nil")
		return
	}

	var loginReq LoginReq
	if err = json.Unmarshal(arg.Data, &loginReq); err != nil {
		log.Error("connect parse LoginReq failed[%s],err:%v", string(arg.Data), err)
		return
	}

	log.Info("receive loginReq:%v", loginReq)

	var key string
	if key, _, err = r.doLogin(arg.Server, &loginReq); err != nil {
		log.Error("connect doLogin failed[%s], err:%v", string(arg.Data), err)
		return
	}

	log.Info("connect doLogin ok[%s]", string(arg.Data))

	reply.Ok = true
	reply.Key = key

	return
}

// Disconnect notice router offline
func (r *RPC) Disconnect(arg *proto.DisconnArg, reply *proto.DisconnReply) (err error) {
	if arg == nil {
		err = ErrDisconnectArgs
		log.Error("Disconnect arg is nil")
		return
	}

	if reply.Has, _, err = r.doLogout(arg.Key, nil); err != nil {
		log.Warn("Disconnect error.key:%s,err:%v", arg.Key, err)
		return
	}
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
		resp, err = r.doHeartbeat(arg.Key, &hbReq)
		reply.Op = define.OP_HEARTBEAT_REPLY

	case define.OP_SEND_MSG:
		var msgReq SendMsgReq
		if err = json.Unmarshal(arg.Data, &msgReq); err != nil {
			return
		}
		resp, err = r.doSendMsg(arg.Key, &msgReq)
		reply.Op = define.OP_SEND_MSG_REPLY

	case define.OP_MSG_SYNC:
		// TODO
		reply.Op = define.OP_MSG_SYNC_REPLY

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

func (r *RPC) doLogin(serverId int32, req *LoginReq) (key string, resp *LoginResp, err error) {
	if req == nil {
		log.Error("doLogin req is nil")
		err = ErrInvalidArgument
		return
	}

	appId := req.UserInfo.AppId
	userId := req.UserInfo.UserId

	if len(appId) == 0 || len(userId) == 0 {
		log.Error("doLogin missing appId(%v) or userId(%v)", appId, userId)
		err = ErrInvalidArgument
		return
	}

	var (
		seq      int32
		session  *Session
		routeKey string
	)

	seq = util.GetTimestampSecond()

	// 连接标识
	key = encodeUserKey(appId, userId, seq)

	// 用户标识 (暂不支持多端登录)
	routeKey = encodeRouteKey(appId, userId)

	if ok := r.auther.Auth(appId, userId, req.Token); ok {
		if session, err = Router.Get(&GetRouteArg{Key: routeKey}); err == nil {
			if session.Seq >= seq {
				// 无效登录
				log.Warn("invalid login.session.Seq %d > seq %d.key:%s", session.Seq, seq, routeKey)
				err = ErrInvalidReq
				return
			}

			// 踢掉
			oldKey := encodeUserKey(appId, userId, session.Seq)
			sPushComet(session.ServerId, oldKey, define.OP_KICKOUT, []byte("{}"))
			log.Warn("send kickout user %s to server %d", oldKey, session.ServerId)
		}

		// 设置router
		if err = Router.Set(&SetRouteArg{Key: routeKey, Session: Session{ServerId: serverId, Seq: seq}}); err != nil {
			log.Error("login set route failed user %s to server %d", routeKey, serverId)
			return
		}
		log.Info("user %s login route at server %d", routeKey, serverId)
	} else {
		log.Error("doLogin auth failed,token=%s, appId:%s, userId:%s", req.Token, appId, userId)
		err = ErrAuthFailed
	}
	return
}

func (r *RPC) doLogout(key string, req *LogoutReq) (has bool, resp LogoutResp, err error) {
	if len(key) == 0 {
		log.Error("doLogout key is empty")
		err = ErrInvalidArgument
		return
	}

	var (
		appId    string
		userId   string
		seq      int32
		routeKey string
		session  *Session
	)

	if appId, userId, seq, err = decodeUserKey(key); err != nil {
		log.Error("doLogout decode key failed:%s.err:%v", key, err)
		err = ErrInvalidArgument
		return
	}

	routeKey = encodeRouteKey(appId, userId)

	if session, err = Router.Get(&GetRouteArg{Key: routeKey}); err == nil {
		if session.Seq != seq {
			log.Warn("invalid logout.session.Seq %d > seq %d.key:%s", session.Seq, seq, routeKey)
			err = ErrInvalidReq
			return
		}
	} else {
		log.Error("doLogout delete session not found:%s.err:%v", key, err)
		has = false
		err = nil
		return
	}

	if has, err = Router.Del(&DelRouteArg{Key: routeKey}); err != nil {
		log.Error("doLogout delete session failed:%s.err:%v", key, err)
	}

	log.Info("doLogout ok.user:%s", key)

	return
}

func (r *RPC) doHeartbeat(key string, hbReq *HeartbeatReq) (resp HeartbeatResp, err error) {
	// if req == nil {
	// 	err = ErrInvalidArgument
	// 	return
	// }

	// if len(key) == 0 {
	// 	err = ErrInvalidArgument
	// 	return
	// }

	// if appId, userId, err = decodeUserKey(key); err != nil {
	// 	err = ErrInvalidArgument
	// 	return
	// }
	return
}

func (r *RPC) doSendMsg(key string, msgReq *SendMsgReq) (resp SendMsgResp, err error) {
	return
}
