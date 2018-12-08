package proto

type SaveMsgArg struct {
	AppId      string
	ChatType   int32
	ChatCode   string
	FromUserId string
	LastMsgId  int64
	MsgData    string
	Tag        string
	CreateTime int64
}

type SaveMsgReply struct {
	MsgId      int64
	PreMsgId   int64
	PreMsgList []MsgData
}

type MsgData struct {
	MsgId      int64  `json:"msgId"`
	PreMsgId   int64  `json:"preMsgId"`
	FromUserId string `json:"fromUserId"`
	MsgData    string `json:"msgData"`
	Tag        string `json:"tag"`
	CreateTime int64  `json:"createTime"`
}

type GetMsgListArg struct {
	AppId    string
	ChatType int32
	ChatCode string

	StartMsgId int64 // 下翻要比该id大，上翻要比该id小
	Direction  int32 // 下翻:0(默认) 上翻:1
	Count      int32
}

type GetMsgListReply struct {
	MsgList []MsgData
}
