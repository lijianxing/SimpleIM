package main

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
	UserInfo UserInfo `json:"user_info"`
}

type HeartbeatResp struct {
}

type SendMsgReq struct {
	UserInfo   UserInfo `json:"user_info"`
	FromUserId string   `json:"from_userid"`

	// 1:user 2:group
	TargetType int    `json:"target_type"`
	TargetId   string `json:"target_id"`

	// 应用消息数据
	// 应用消息元数据(可选, 例如存放发送方的一些附加信息,昵称/头像url什么的)
	MsgMeta string `json:"msg_meta"`
	MsgData string `json:"msg_data"`

	// 应用消息标记(可选填, 可用于搜索过滤, 例如存放咨询订单号)
	Tag string `json:"tag"`
}

type SendMsgResp struct {
	MsgId int64 `json:"msg_id"`
}

type LogoutReq struct {
	UserInfo UserInfo `json:"user_info"`
}

type LogoutResp struct {
}
