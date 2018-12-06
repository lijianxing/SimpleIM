package proto

type SaveMsgArg struct {
	AppId      string
	ChatType   int
	ChatCode   string
	FromUserId string
	MsgData    string
	Tag        string
	CreateTime int64
}

type SaveMsgReply struct {
	ChatMsgId    int64
	ChatPreMsgId int64
}

type MsgData struct {
	MsgId      int64  `json:"msgid"`
	PreMsgId   int64  `json:"pre_msgid"`
	FromUserId string `json:"from_user_id"`
	MsgData    string `json:"msg_data"`
	Tag        string `json:"msg_tag"`
	CreateTime int64  `json:"create_time"`
}

type GetMsgListArg struct {
	AppId    string
	ChatType int
	ChatCode string

	StartMsgId int64 // 下翻要比该id大，上翻要比该id小
	Direction  int   // 下翻:0(默认) 上翻:1
	Count      int
}

type GetMsgListReply struct {
	MsgList []MsgData
}
