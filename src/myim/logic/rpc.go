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

	if reply.Has, _, err = r.doLogout(arg.Server, arg.Key, nil); err != nil {
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
		// 暂时不需要请求/响应数据
		op = define.OP_HEARTBEAT_REPLY
		_, err = r.doHeartbeat(arg.Server, arg.Key, nil)

	case define.OP_SEND_MSG:
		var msgReq SendMsgReq
		if err = json.Unmarshal(arg.Data, &msgReq); err != nil {
			return
		}
		resp, err = r.doSendMsg(arg.Server, arg.Key, &msgReq)
		op = define.OP_SEND_MSG_REPLY

	case define.OP_MSG_SYNC:
		// TODO
		op = define.OP_MSG_SYNC_REPLY

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

func (r *RPC) doLogout(serverId int32, key string, req *LogoutReq) (has bool, resp *LogoutResp, err error) {
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

func (r *RPC) doHeartbeat(serverId int32, key string, req *HeartbeatReq) (resp *HeartbeatResp, err error) {
	if len(key) == 0 {
		err = ErrInvalidArgument
		return
	}

	log.Debug("receive heartbeat msg.serverId:%d, key:%s", serverId, key)

	var (
		appId  string
		userId string
		seq    int32
	)

	if appId, userId, seq, err = decodeUserKey(key); err != nil {
		log.Error("decode user key failed, key:%s, err:%v", key, err)
		err = ErrInvalidArgument
		return
	}

	// 路由维护
	if err = refreshRoute(serverId, appId, userId, seq); err != nil {
		log.Error("hearbeat refresh route failed.key:%s, err:%v", key, err)
	}

	// always succ
	err = nil
	resp = nil
	return
}

func (r *RPC) doSendMsg(serverId int32, key string, req *SendMsgReq) (resp *SendMsgResp, err error) {
	if len(key) == 0 || req == nil {
		err = ErrInvalidArgument
		return
	}

	log.Debug("receive sendmsg req:%v", *req)

	var (
		appId  string
		userId string
		seq    int32

		// sessions
		targetUserIds  []string
		targetSessions []*Session
	)

	if appId, userId, seq, err = decodeUserKey(key); err != nil {
		log.Error("decode user key failed, key:%s, err:%v", key, err)
		err = ErrInvalidArgument
		return
	}

	// 路由维护
	if err = refreshRoute(serverId, appId, userId, seq); err != nil {
		log.Error("sendmsg refresh route failed.key:%s, err:%v", key, err)
	}

	// save msg to db and get msgid
	// TODO
	msgId := int64(100)

	// reply to sender
	resp = &SendMsgResp{
		MsgId: msgId,
	}

	// msg notify
	if req.TargetType == define.TARGET_USER {
		targetUserIds = append(targetUserIds, req.TargetId)
	} else if req.TargetType == define.TARGET_GROUP {
		// TODO: get group members
	} else {
		// error
	}

	// get target user sessions
	args := make([]*GetRouteArg, len(targetUserIds))
	for i, targetUserId := range targetUserIds {
		args[i] = &GetRouteArg{
			Key: encodeRouteKey(appId, targetUserId),
		}
	}
	if targetSessions, err = Router.MGet(args); err != nil {
		log.Error("get target user sessions failed.err:%v", err)
		return
	}
	//implied: len(targetSessions) == len(targetUserIds)

	// 按server分组
	serverKeyMap := make(map[int32][]string)
	for i, session := range targetSessions {
		if session == nil {
			continue
		}
		keys := serverKeyMap[session.ServerId]
		keys = append(keys, encodeUserKey(appId, targetUserIds[i], session.Seq))
		serverKeyMap[session.ServerId] = keys
	}

	// 发送msg
	op := define.OP_MSG_NOTIFY
	notify := MsgNotify{
		FromUserId: userId,
		TargetType: req.TargetType,
		TargetId:   req.TargetId,
		MsgMeta:    req.MsgMeta,
		MsgData:    req.MsgData,
		Tag:        req.Tag,
	}
	data, _ := json.Marshal(notify)

	for srvId, userIds := range serverKeyMap {
		mPushComet(srvId, userIds, op, data)
	}

	return
}

func refreshRoute(serverId int32, appId string, userId string, seq int32) (err error) {

	var session *Session
	routeKey := encodeRouteKey(appId, userId)

	if session, err = Router.Get(&GetRouteArg{Key: routeKey}); err == nil {
		if session.Seq > seq {
			log.Warn("invalid refresh route.session.Seq %d > seq %d.key:%s", session.Seq, seq, routeKey)
			err = ErrInvalidReq
			return
		}
	}

	// 路由更新
	if err = Router.Set(&SetRouteArg{Key: routeKey, Session: Session{ServerId: serverId, Seq: seq}}); err != nil {
		log.Error("login set route failed user %s to server %d", routeKey, serverId)
		return
	}
	log.Debug("login refresh route ok.key:%s, server:%d", routeKey, serverId)
	return
}
