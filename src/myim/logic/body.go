package main

import "myim/libs/proto"

type UserInfo struct {
	AppId  string `json:"app_id"`
	UserId string `json:"user_id"`
}

type LoginReq struct {
	Token    string   `json:"token"`
	UserInfo UserInfo `json:"user_info"`
}

type LoginResp struct {
}

type HeartbeatReq struct {
}

type HeartbeatResp struct {
}

type SendMsgReq struct {
	// 1:user 2:group
	TargetType int    `json:"target_type"`
	TargetId   string `json:"target_id"`

	// 应用消息数据
	MsgData string `json:"msg_data"`

	// 应用消息标记(可选填, 可用于搜索过滤, 例如存放咨询订单号)
	Tag string `json:"tag"`
}

type SendMsgResp struct {
	// 会话 msgid
	ChatMsgId    int64 `json:"chat_msgid"`
	ChatPreMsgId int64 `json:"chat_premsgid"`
}

type MsgNotify struct {
	// 1:user 2:group
	TargetType int    `json:"target_type"`
	TargetId   string `json:"target_id"`

	Msg proto.MsgData `json:"msg"`
}

type MsgNotifyAck struct {
}

type SyncMsgReq struct {
	// 1:user 2:group
	TargetType int    `json:"target_type"`
	TargetId   string `json:"target_id"`

	Direction  int   `json:"direction"`
	StartMsgId int64 `json:"start_msgid"`
	Count      int   `json:"count"`
}

type SyncMsgResp struct {
	MsgList []proto.MsgData `json:"msg_list"`
}

type LogoutReq struct {
}

// useless
type LogoutResp struct {
}
