package proto

type PutArg struct {
	UserId string
	Server int32
}

type PutReply struct {
}

type DelArg struct {
	UserId string
	Seq    int32
}

type DelReply struct {
	Has bool
}

type GetArg struct {
	UserId string
}

type GetReply struct {
	Seqs    []int32
	Servers []int32
}

type GetAllReply struct {
	UserIds  []string
	Sessions []*GetReply
}

type MGetArg struct {
	UserIds []string
}

type MGetReply struct {
	UserIds  []string
	Sessions []*GetReply
}
