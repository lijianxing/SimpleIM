package proto

type NoArg struct {
}

type NoReply struct {
}

type PushMsgArg struct {
	Key string
	P   Proto
}

type PushMsgsArg struct {
	Key    string
	PMArgs []*PushMsgArg
}

type PushMsgsReply struct {
	Index int32
}

type MPushMsgArg struct {
	Keys []string
	P    Proto
}

type MPushMsgReply struct {
	Index int32
}

type MPushMsgsArg struct {
	PMArgs []*PushMsgArg
}

type MPushMsgsReply struct {
	Index int32
}

type BoardcastArg struct {
	P Proto
}
