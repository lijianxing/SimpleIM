package define

const (
	// login
	OP_LOGIN       = int32(0)
	OP_LOGIN_REPLY = int32(1)

	// heartbeat
	OP_HEARTBEAT       = int32(2)
	OP_HEARTBEAT_REPLY = int32(3)

	// send messgae
	OP_SEND_MSG       = int32(4)
	OP_SEND_MSG_REPLY = int32(5)

	// push msg notify
	OP_MSG_NOTIFY     = int32(6)
	OP_MSG_NOTIFY_ACK = int32(7)

	// msg sync
	OP_MSG_SYNC       = int32(8)
	OP_MSG_SYNC_REPLY = int32(9)

	// logout
	OP_LOGOUT       = int32(10)
	OP_LOGOUT_REPLY = int32(11)

	// test
	OP_TEST       = int32(12)
	OP_TEST_REPLY = int32(13)

	// raw message
	OP_RAW = int32(21)

	// 内部使用
	// proto
	OP_PROTO_READY  = int32(22)
	OP_PROTO_FINISH = int32(23)
)
