package main

import "myim/libs/proto"

type Ring struct {
	curr int
	num  int
	data []*proto.MsgData
}

func NewRing(num int) *Ring {
	r := new(Ring)
	r.Init(num)
	return r
}

func (r *Ring) Init(num int) {
	r.data = make([]*proto.MsgData, num)
	r.num = num
}

func (r *Ring) AddMsg(msg *proto.MsgData) {
	r.data[r.curr] = msg
	r.curr = (r.curr + 1) % r.num
}

func (r *Ring) GetFirst() (msg *proto.MsgData) {
	// empty
	if r.curr == 0 && r.data[0] == nil {
		return nil
	}
	// not full
	if r.data[r.curr] == nil {
		return r.data[0]
	}
	// full
	return r.data[r.curr]
}

func (r *Ring) GetLast() (msg *proto.MsgData) {
	// empty
	if r.curr == 0 && r.data[0] == nil {
		return nil
	}
	// not full
	if r.data[r.curr] == nil {
		return r.data[r.curr-1]
	}
	// full
	return r.data[(r.curr-1+r.num)%r.num]
}

func (r *Ring) GetMsgList() (msgList []*proto.MsgData) {
	// empty
	if r.curr == 0 && r.data[0] == nil {
		return nil
	}
	// not full
	if r.data[r.curr] == nil {
		msgList = make([]*proto.MsgData, r.curr)
		copy(msgList, r.data[:r.curr])
		return
	}
	// full
	for i := 0; i < r.num; i++ {
		msgList = append(msgList, r.data[(r.curr+i)%r.num])
	}
	return
}

func (r *Ring) Len() int {
	// empty
	if r.curr == 0 && r.data[0] == nil {
		return 0
	}
	// not full
	if r.data[r.curr] == nil {
		return r.curr
	}
	// full
	return r.num
}

func (r *Ring) Num() int {
	return r.num
}

// 会话最近消息队列 (最近N个)
type ChatQueue struct {
	msgQueue *Ring
}

func NewChatQueue(num int) *ChatQueue {
	q := new(ChatQueue)
	q.Init(num)
	return q
}

func (cq *ChatQueue) Init(num int) {
	cq.msgQueue = NewRing(num)
}

func (cq *ChatQueue) AddMsg(msg *proto.MsgData) (preMsgId int64) {
	preMsg := cq.msgQueue.GetLast()
	preMsgId = msg.PreMsgId
	if preMsg != nil {
		preMsgId = preMsg.MsgId
	}
	msg.PreMsgId = preMsgId
	cq.msgQueue.AddMsg(msg)
	return preMsgId
}

func (cq *ChatQueue) GetMsgList() (msgList []*proto.MsgData) {
	return cq.msgQueue.GetMsgList()
}

func (cq *ChatQueue) GetFirst() *proto.MsgData {
	return cq.msgQueue.GetFirst()
}

func (cq *ChatQueue) GetLast() *proto.MsgData {
	return cq.msgQueue.GetLast()
}
