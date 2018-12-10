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
		log.Error("connect parse LoginReq failed.data=%s, serverId:%d, err:%v", string(arg.Data), arg.Server, err)
		return
	}

	log.Info("receive serverId:%d, LoginReq:%v", arg.Server, loginReq)

	var key string
	var resp *LoginResp
	if key, resp, err = r.doLogin(arg.Server, &loginReq); err != nil {
		log.Error("connect doLogin failed. server:%d, LoginReq:%v, err:%v", arg.Server, loginReq, err)
		return
	}

	reply.Ok = true
	reply.Key = key
	reply.Heartbeat = Conf.ClientHeartbeat

	if resp != nil {
		if reply.Data, err = json.Marshal(resp); err != nil {
			log.Error("Login marshal data failed.resp:%v, err:%v", resp, err)
			return
		}
	}

	log.Info("connect doLogin ok.LoginReq:%v, reply:%v", loginReq, reply)

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
		log.Warn("Disconnect error.server:%d, key:%s, err:%v", arg.Server, arg.Key, err)
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
		op = define.OP_HEARTBEAT_REPLY
		_, err = r.doHeartbeat(arg.Server, arg.Key, nil)

	case define.OP_SEND_MSG:
		var msgReq SendMsgReq
		if err = json.Unmarshal(arg.Data, &msgReq); err != nil {
			return
		}
		op = define.OP_SEND_MSG_REPLY
		resp, err = r.doSendMsg(arg.Server, arg.Key, &msgReq)

	case define.OP_MSG_SYNC:
		op = define.OP_MSG_SYNC_REPLY
		var syncReq SyncMsgReq
		if err = json.Unmarshal(arg.Data, &syncReq); err != nil {
			return
		}
		op = define.OP_MSG_SYNC_REPLY
		resp, err = r.doSyncMsg(arg.Server, arg.Key, &syncReq)

	case define.OP_MSG_NOTIFY_ACK:
		log.Debug("recive msg notify ack.req:%v", *arg)

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
			log.Error("Operate marshal data failed.resp:%v, err:%v", resp, err)
			return
		}
	}
	reply.Op = op

	return
}

///////////////  logic /////////////

func (r *RPC) doLogin(serverId int32, req *LoginReq) (key string, resp *LoginResp, err error) {
	if req == nil {
		log.Error("doLogin req is nil, serverId:%d", serverId)
		err = ErrInvalidArgument
		return
	}

	appId := req.UserInfo.AppId
	userId := req.UserInfo.UserId

	if len(appId) == 0 || len(userId) == 0 {
		log.Error("doLogin invalid params. serverId:%d, req:%v", serverId, *req)
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

	if ok := r.auther.Auth(appId, userId, req.Token); !ok {
		log.Error("doLogin auth failed,token=%s, appId:%s, userId:%s", req.Token, appId, userId)
		err = ErrAuthFailed
		return
	}

	if session, err = getRoute(&GetRouteArg{Key: routeKey}); err == nil {
		if session.Seq >= seq {
			// 无效登录
			log.Warn("invalid login.session.Seq %d > seq %d.key:%s", session.Seq, seq, routeKey)
			err = ErrInvalidReq
			return
		}

		// 踢掉
		oldKey := encodeUserKey(appId, userId, session.Seq)
		sPushComet(session.ServerId, oldKey, define.OP_KICKOUT, []byte("{}"))
		log.Warn("send kickout user %s at server %d", oldKey, session.ServerId)
	}

	// 设置router
	if err = setRoute(&SetRouteArg{Key: routeKey, Session: Session{ServerId: serverId, Seq: seq}}); err != nil {
		log.Error("login set route failed user %s at server %d", routeKey, serverId)
		return
	}
	log.Info("user %s login route at server %d", routeKey, serverId)

	resp = &LoginResp{
		RetCode:   define.RC_OK,
		Heartbeat: int32(Conf.ClientHeartbeat),
	}

	return
}

func (r *RPC) doLogout(serverId int32, key string, req *LogoutReq) (has bool, resp *LogoutResp, err error) {
	if len(key) == 0 {
		log.Error("doLogout key is empty.serverId:%d", serverId)
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
		log.Error("doLogout decode key failed:%s. serverId:%d, err:%v", key, serverId, err)
		err = ErrInvalidArgument
		return
	}

	routeKey = encodeRouteKey(appId, userId)

	if session, err = getRoute(&GetRouteArg{Key: routeKey}); err == nil {
		if session.Seq != seq {
			log.Warn("invalid logout.session.Seq %d > seq %d.key:%s", session.Seq, seq, routeKey)
			err = ErrInvalidReq
			return
		}
	} else {
		log.Error("doLogout delete session not found:%s.serverId:%d, err:%v", key, serverId, err)
		has = false
		err = nil
		return
	}

	if has, err = delRoute(&DelRouteArg{Key: routeKey}); err != nil {
		log.Error("doLogout delete session failed:%s.serverId:%d, err:%v", key, serverId, err)
	}

	log.Info("doLogout ok.user:%s, serverId:%d", key, serverId)

	return
}

func (r *RPC) doHeartbeat(serverId int32, key string, req *HeartbeatReq) (resp *HeartbeatResp, err error) {
	if len(key) == 0 {
		log.Error("doHeartbeat key is empty.serverId:%d", serverId)
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

func (r *RPC) doSendMsg(serverId int32, key string, req *SendMsgReq) (*SendMsgResp, error) {
	if len(key) == 0 || req == nil || req.TargetId == "" {
		log.Error("doSendMsg invalid params.serverId:%d, key:%s, req:%v", serverId, key, req)
		return nil, ErrInvalidArgument
	}

	log.Debug("receive sendmsg req:%v", *req)

	var (
		appId  string
		userId string
		seq    int32

		// sessions
		targetUserIds  []string
		targetSessions []*Session

		// group user
		groupUsers []string

		// saveMsg
		saveMsgReply *proto.SaveMsgReply

		chatCode string
		now      = util.GetTimestampMillSec()

		resp *SendMsgResp

		errResp *SendMsgResp = &SendMsgResp{RetCode: define.RC_ERROR}
		err     error
	)

	if appId, userId, seq, err = decodeUserKey(key); err != nil {
		log.Error("decode user key failed, key:%s, err:%v", key, err)
		return nil, ErrInvalidArgument
	}

	// 路由维护
	if err = refreshRoute(serverId, appId, userId, seq); err != nil {
		log.Error("sendmsg refresh route failed.key:%s, err:%v", key, err)
	}

	// msg notify
	if req.TargetType == define.TARGET_USER {
		// 单聊
		if req.TargetId != userId {
			targetUserIds = append(targetUserIds, req.TargetId)
			chatCode = encodeSinleSessionCode(userId, req.TargetId)
		}
	} else if req.TargetType == define.TARGET_GROUP {
		// 群聊
		chatCode = encodeGroupSessionCode(req.TargetId)

		groupUsers, err = getGroupUsers(&GetGroupUsersArg{
			AppId:     appId,
			GroupCode: req.TargetId,
		})
		if err != nil {
			log.Error("get group user failed, appId:%s, groupCode:%, err:%v", appId, req.TargetId, err)
			return errResp, nil
		}

		for _, groupUser := range groupUsers {
			// 排除自身
			if groupUser == userId {
				continue
			}
			targetUserIds = append(targetUserIds, groupUser)
		}
	} else {
		log.Error("unknown target type.target:%d, appid:%s", req.TargetType, appId)
		return nil, ErrUnknownTarget
	}

	// save msg to db and get msgid
	msg := &proto.SaveMsgArg{
		AppId:      appId,
		ChatType:   req.TargetType,
		ChatCode:   chatCode,
		FromUserId: userId,
		LastMsgId:  req.LastMsgId,
		MsgData:    req.MsgData,
		Tag:        req.Tag,
		CreateTime: now,
	}
	if saveMsgReply, err = saveMsg(msg); err != nil {
		log.Error("save msg failed.msg:%v, err:%v", *msg, err)
		return errResp, nil
	}

	// reply to sender
	resp = &SendMsgResp{
		RetCode:    define.RC_OK,
		MsgId:      saveMsgReply.MsgId,
		PreMsgId:   saveMsgReply.PreMsgId,
		PreMsgList: saveMsgReply.PreMsgList,
	}

	//////////// msg notify ///////////////
	if len(targetUserIds) == 0 {
		// nothing to do
		log.Debug("no target user.appid:%s, targetType:%d, targetId:%s",
			appId, req.TargetType, req.TargetId)
		return resp, nil
	}

	// get target user sessions
	args := make([]*GetRouteArg, len(targetUserIds))
	for i, targetUserId := range targetUserIds {
		args[i] = &GetRouteArg{
			Key: encodeRouteKey(appId, targetUserId),
		}
	}
	if targetSessions, err = mGetRoute(args); err != nil {
		log.Error("get target user sessions failed.args:%v, err:%v", args, err)
		return resp, nil
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
		TargetType: req.TargetType,
		TargetId:   req.TargetId,
		Msg: proto.MsgData{
			FromUserId: userId,
			MsgId:      saveMsgReply.MsgId,
			PreMsgId:   saveMsgReply.PreMsgId,
			MsgData:    req.MsgData,
			Tag:        req.Tag,
			CreateTime: now,
		},
	}
	data, _ := json.Marshal(notify)

	for srvId, userIds := range serverKeyMap {
		mPushComet(srvId, userIds, op, data)
	}

	return resp, nil
}

func (r *RPC) doSyncMsg(serverId int32, key string, req *SyncMsgReq) (*SyncMsgResp, error) {
	if len(key) == 0 || req == nil || req.TargetId == "" {
		log.Error("doSyncMsg invalid params.serverId:%d, key:%s, req:%v", serverId, key, req)
		return nil, ErrInvalidArgument
	}

	log.Debug("receive syncmsg req:%v", *req)

	var (
		appId  string
		userId string
		seq    int32

		chatCode string
		resp     *SyncMsgResp
		errResp  *SyncMsgResp = &SyncMsgResp{RetCode: define.RC_ERROR}
		reply    *proto.GetMsgListReply
		err      error
	)

	if appId, userId, seq, err = decodeUserKey(key); err != nil {
		log.Error("decode user key failed, key:%s, err:%v", key, err)
		return nil, ErrInvalidArgument
	}

	// 路由维护
	if err = refreshRoute(serverId, appId, userId, seq); err != nil {
		log.Error("sendmsg refresh route failed.key:%s, err:%v", key, err)
	}

	if req.TargetType == define.TARGET_USER {
		// 单聊
		if req.TargetId != userId {
			chatCode = encodeSinleSessionCode(userId, req.TargetId)
		}
	} else if req.TargetType == define.TARGET_GROUP {
		// 群聊
		chatCode = encodeGroupSessionCode(req.TargetId)
	} else {
		log.Error("unknown target type.target:%d, appid:%s", req.TargetType, appId)
		return nil, ErrUnknownTarget
	}

	if chatCode == "" {
		log.Error("get chat code failed.req:%v", *req)
		return nil, ErrInvalidArgument
	}

	reply, err = getMsgList(&proto.GetMsgListArg{
		AppId:      appId,
		ChatType:   req.TargetType,
		ChatCode:   chatCode,
		StartMsgId: req.StartMsgId,
		Direction:  req.Direction,
		Count:      req.Count,
	})
	if err != nil {
		log.Error("sync get msg list failed.req:%v, err:%v", *req, err)
		return errResp, nil
	}

	resp = &SyncMsgResp{
		RetCode: define.RC_OK,
		MsgList: reply.MsgList,
	}
	log.Debug("getmsglist rpc reply:%v", resp.MsgList)

	return resp, nil
}

func refreshRoute(serverId int32, appId string, userId string, seq int32) (err error) {

	var session *Session
	routeKey := encodeRouteKey(appId, userId)

	if session, err = getRoute(&GetRouteArg{Key: routeKey}); err == nil {
		if session.Seq > seq {
			log.Warn("invalid refresh route.session.Seq %d > seq %d.key:%s", session.Seq, seq, routeKey)
			err = ErrInvalidReq
			return
		}
	}

	// 路由更新
	if err = setRoute(&SetRouteArg{Key: routeKey, Session: Session{ServerId: serverId, Seq: seq}}); err != nil {
		log.Error("login set route failed user %s to server %d", routeKey, serverId)
		return
	}
	log.Debug("login refresh route ok.key:%s, server:%d", routeKey, serverId)
	return
}
