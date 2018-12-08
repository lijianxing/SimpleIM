package main

import "myim/libs/proto"

type UserInfo struct {
	AppId  string `json:"appId"`
	UserId string `json:"userId"`
}

type LoginReq struct {
	Token    string   `json:"token"`
	UserInfo UserInfo `json:"userInfo"`
}

type LoginResp struct {
	RetCode int32 `json:"retCode"`
	// 心跳超时,单位秒
	Heartbeat int32 `json:"heartbeat"`
}

type HeartbeatReq struct {
}

type HeartbeatResp struct {
}

type SendMsgReq struct {
	// 1:user 2:group
	TargetType int32  `json:"targetType"`
	TargetId   string `json:"targetId"`
	LastMsgId  int64  `json:"lastMsgId"`

	// 应用消息数据
	MsgData string `json:"msgData"`

	// 应用消息标记(可选填, 可用于搜索过滤, 例如存放咨询订单号)
	Tag string `json:"tag"`
}

type SendMsgResp struct {
	RetCode    int32           `json:"retCode"`
	MsgId      int64           `json:"msgId"`
	PreMsgId   int64           `json:"preMsgId"`
	PreMsgList []proto.MsgData `json:"preMsgList"`
}

type MsgNotify struct {
	// 1:user 2:group
	TargetType int32  `json:"targetType"`
	TargetId   string `json:"targetId"`

	Msg proto.MsgData `json:"msg"`
}

type MsgNotifyAck struct {
}

type SyncMsgReq struct {
	// 1:user 2:group
	TargetType int32  `json:"targetType"`
	TargetId   string `json:"targetId"`

	Direction  int32 `json:"direction"`
	StartMsgId int64 `json:"startMsgId"`
	Count      int32 `json:"count"`
}

type SyncMsgResp struct {
	RetCode int32           `json:"retCode"`
	MsgList []proto.MsgData `json:"msgList"`
}

type LogoutReq struct {
}

// useless
type LogoutResp struct {
}
