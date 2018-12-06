package proto

type ConnArg struct {
	Server int32  // serverId
	Data   []byte // body数据
}

type ConnReply struct {
	Ok        bool   // 是否允许连接
	Key       string // 连接唯一标识
	Heartbeat int    // 心跳超时(秒)
}

type OperArg struct {
	Server int32
	Key    string
	SeqId  int32
	Op     int32
	Data   []byte
}

type OperReply struct {
	Op   int32
	Data []byte
}

type DisconnArg struct {
	Server int32
	Key    string
}

type DisconnReply struct {
	Has bool
}
