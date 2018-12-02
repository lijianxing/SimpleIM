package main

import (
	"sync/atomic"
	"time"
)

type Stat struct {
	// msg
	MsgSucceeded uint64 `json:"msg_succeeded"`
	MsgFailed    uint64 `json:"msg_failed"`

	// sync
	SyncTimes uint64 `json:"sync_times"`

	// speed
	SpeedMsgSecond uint64 `json:"speed_msg_second"`

	// nodes
	RouterNodes map[string]string `json:"router_nodes"`

	// messages
	PushMsg      uint64 `json:"push_msg"`
	BroadcastMsg uint64 `json:"broadcast_msg"`

	// miss
	PushMsgFailed      uint64 `json:"push_msg_failed"`
	BroadcastMsgFailed uint64 `json:"broadcast_msg_failed"`
}

func NewStat() *Stat {
	s := new(Stat)
	go s.procSpeed()
	return s
}

func (s *Stat) Info() *Stat {
	return s
}

func (s *Stat) Reset() {
	atomic.StoreUint64(&s.MsgSucceeded, 0)
	atomic.StoreUint64(&s.MsgFailed, 0)
	atomic.StoreUint64(&s.SyncTimes, 0)

	atomic.StoreUint64(&s.PushMsg, 0)
	atomic.StoreUint64(&s.BroadcastMsg, 0)
	atomic.StoreUint64(&s.PushMsgFailed, 0)
	atomic.StoreUint64(&s.BroadcastMsgFailed, 0)
}

func (s *Stat) procSpeed() {
	var (
		timer   = uint64(5) // diff 5s
		lastMsg uint64
	)
	for {
		s.SpeedMsgSecond = (atomic.LoadUint64(&s.MsgSucceeded) - lastMsg) / timer
		lastMsg = s.MsgSucceeded
		time.Sleep(time.Duration(timer) * time.Second)
	}
}

func (s *Stat) IncrMsgSucceeded() {
	atomic.AddUint64(&s.MsgSucceeded, 1)
}

func (s *Stat) IncrMsgFailed() {
	atomic.AddUint64(&s.MsgFailed, 1)
}

func (s *Stat) IncrSyncTimes() {
	atomic.AddUint64(&s.SyncTimes, 1)
}

func (s *Stat) IncrPushMsg() {
	atomic.AddUint64(&s.PushMsg, 1)
}

func (s *Stat) IncrBroadcastMsg() {
	atomic.AddUint64(&s.BroadcastMsg, 1)
}

func (s *Stat) IncrPushMsgFailed() {
	atomic.AddUint64(&s.PushMsgFailed, 1)
}

func (s *Stat) IncrBroadcastMsgFailed() {
	atomic.AddUint64(&s.BroadcastMsgFailed, 1)
}
