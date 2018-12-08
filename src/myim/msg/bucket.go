package main

import (
	"myim/libs/define"
	"myim/libs/proto"
	"sync"
	"time"

	log "github.com/thinkboy/log4go"
)

const (
	DEFAULT_MSG_NUM = 10
	MAX_MSG_NUM     = 50
)

type ChatData struct {
	appId    string
	chatType int32
	chatCode string

	// 读消息副本, 读写可并发
	msgMutex     sync.Mutex
	msgListReady bool
	msgList      []*proto.MsgData // 消息列表副本

	// 写串行, 保证cache和db一致
	chatMutex      sync.Mutex
	chatQueueReady bool
	chatQueue      *ChatQueue
}

func NewChatData(appId string, chatType int32, chatCode string, msgs int) *ChatData {
	cd := new(ChatData)
	cd.chatQueue = NewChatQueue(msgs)

	cd.appId = appId
	cd.chatType = chatType
	cd.chatCode = chatCode

	return cd
}

type Bucket struct {
	bLock   sync.RWMutex
	chat    int // bucket chat init num
	msgs    int
	chats   map[string]*ChatData // chatid->chatdata
	cleaner *Cleaner             // bucket map cleaner
	expire  time.Duration
}

func NewBucket(msgs int, chat int, expire time.Duration) *Bucket {
	b := new(Bucket)
	b.chats = make(map[string]*ChatData, chat)
	b.msgs = msgs
	b.chat = chat
	b.cleaner = NewCleaner(chat)
	b.expire = expire
	go b.clean()
	return b
}

func (b *Bucket) SaveMsg(chatKey string, arg *proto.SaveMsgArg, reply *proto.SaveMsgReply) (err error) {
	var (
		cd         *ChatData
		ok         bool
		msgId      int64
		preMsgId   int64
		msgList    []*proto.MsgData
		preMsgList []proto.MsgData
	)
	b.bLock.Lock()
	if cd, ok = b.chats[chatKey]; !ok {
		cd = NewChatData(arg.AppId, arg.ChatType, arg.ChatCode, b.msgs)
		b.chats[chatKey] = cd
	}
	b.bLock.Unlock()

	// lru
	b.cleaner.PushFront(chatKey, b.expire)

	// 消息存储
	cd.chatMutex.Lock()

	if !cd.chatQueueReady {
		log.Debug("chat queue not ready, load from db.chatKey:%s", chatKey)

		// load last n msgs from db
		msgList, err = getMsgListDb(&proto.GetMsgListArg{
			AppId:      arg.AppId,
			ChatType:   arg.ChatType,
			ChatCode:   arg.ChatCode,
			StartMsgId: 0, // no start msg id
			Direction:  define.DIRECTION_FORWARD,
			Count:      int32(b.msgs),
		})
		if err != nil {
			cd.chatMutex.Unlock()
			log.Error("load chat queue msg from db failed.chatKey:%s, err:%v", chatKey, err)
			return
		}

		for i := 0; i < len(msgList); i++ {
			cd.chatQueue.AddMsg(msgList[i])
		}

		// add to chat queue
		cd.chatQueueReady = true
	}

	if arg.LastMsgId > 0 {
		cacheMsgs := cd.chatQueue.GetMsgList()
		cacheMsgLen := len(cacheMsgs)
		if cacheMsgLen > 0 {
			idx := -1
			for i := 0; i < cacheMsgLen; i++ {
				if cacheMsgs[i].MsgId == arg.LastMsgId {
					idx = i
					break
				}
			}
			if idx >= 0 {
				for i := idx + 1; i < cacheMsgLen; i++ {
					preMsgList = append(preMsgList, *cacheMsgs[i])
				}
				if len(preMsgList) > 0 {
					log.Warn("found client lost msg.chatKey:%s, len=%d", chatKey, len(preMsgList))
				}
			} else {
				// msg gap is larger than cache msg, just reply error let client do msg sync
				cd.chatMutex.Unlock()
				err = ErrLostTooManyMsg
				log.Error("client lost too many msg.chatKey:%s, startMsgId:%d, cacheStartMsgId:%d",
					chatKey, arg.LastMsgId, cacheMsgs[0].MsgId)
				return
			}
		}
	}

	// save msg to db
	msgId, err = saveMsgDb(arg)
	if err != nil {
		cd.chatMutex.Unlock()
		log.Error("save msg to db failed.chatkey:%s, msg:%v", chatKey, *arg)
		return
	}

	// add msg to chatqueue
	preMsgId = cd.chatQueue.AddMsg(&proto.MsgData{
		MsgId:      msgId,
		PreMsgId:   0, // chatqueue 回填
		FromUserId: arg.FromUserId,
		MsgData:    arg.MsgData,
		Tag:        arg.Tag,
		CreateTime: arg.CreateTime,
	})

	reply.MsgId = msgId
	reply.PreMsgId = preMsgId
	reply.PreMsgList = preMsgList

	msgList = cd.chatQueue.GetMsgList()

	cd.chatMutex.Unlock()

	// update msg list cache
	cd.msgMutex.Lock()
	cd.msgList = msgList
	cd.msgListReady = true
	cd.msgMutex.Unlock()

	return
}

func (b *Bucket) GetMsgList(chatKey string, arg *proto.GetMsgListArg, reply *proto.GetMsgListReply) (err error) {
	var (
		cd      *ChatData
		ok      bool
		msgList []*proto.MsgData
	)

	b.bLock.Lock()
	if cd, ok = b.chats[chatKey]; !ok {
		cd = NewChatData(arg.AppId, arg.ChatType, arg.ChatCode, b.msgs)
		b.chats[chatKey] = cd
	}
	b.bLock.Unlock()

	// lru
	b.cleaner.PushFront(chatKey, b.expire)

	cd.msgMutex.Lock()

	if !cd.msgListReady {
		log.Debug("msg list not ready, load from db.chatKey:%s", chatKey)

		// load last n msgs from db
		msgList, err = getMsgListDb(&proto.GetMsgListArg{
			AppId:      arg.AppId,
			ChatType:   arg.ChatType,
			ChatCode:   arg.ChatCode,
			StartMsgId: 0, // no start msg id
			Direction:  define.DIRECTION_FORWARD,
			Count:      int32(b.msgs),
		})
		if err != nil {
			cd.msgMutex.Unlock()
			log.Error("load msglist from db failed.chatKey:%s, err:%v", chatKey, err)
			return
		}

		cd.msgList = msgList

		cd.msgListReady = true
	}

	if arg.Count <= 0 {
		arg.Count = DEFAULT_MSG_NUM
	} else if arg.Count > MAX_MSG_NUM {
		arg.Count = MAX_MSG_NUM
	}

	msgLen := len(cd.msgList)

	// find cache
	cacheHit := false
	start := 0
	count := int(arg.Count)

	if msgLen == 0 {
		// no msg right now, just return empty list
		cacheHit = true
		start = 0
		count = 0
	} else if arg.Direction == define.DIRECTION_BACKWARD {
		// 上翻
		if arg.StartMsgId > 0 {
			endIdx := -1
			for i := msgLen - 1; i >= 0; i-- {
				if cd.msgList[i].MsgId < arg.StartMsgId {
					endIdx = i
					break
				}
			}
			if endIdx >= 0 {
				if endIdx+1 >= int(arg.Count) {
					// 数据足够
					cacheHit = true
					start = endIdx + 1 - int(arg.Count)
					count = int(arg.Count)
				} else if cd.msgList[0].PreMsgId == 0 {
					// 数据不够但到头了
					cacheHit = true
					start = 0
					count = endIdx + 1
				}
			} else if cd.msgList[0].PreMsgId == 0 {
				// 到头了没数据
				cacheHit = true
				start = 0
				count = 0
			}
		} else {
			if int(arg.Count) <= msgLen {
				cacheHit = true
				start = msgLen - int(arg.Count)
				count = int(arg.Count)
			} else if cd.msgList[0].PreMsgId == 0 {
				// 没有前置消息
				cacheHit = true
				start = 0
				count = msgLen
			}
		}
	} else {
		// 下翻
		if arg.StartMsgId > 0 {
			startIdx := -1
			for i := 0; i < msgLen; i++ {
				if cd.msgList[i].MsgId > arg.StartMsgId {
					startIdx = i
					break
				}
			}
			if startIdx == 0 && cd.msgList[0].PreMsgId == 0 || startIdx > 0 {
				// hit
				cacheHit = true
				start = startIdx
				count = int(arg.Count)
			} else if startIdx < 0 {
				// 没有新的数据了
				cacheHit = true
				start = 0
				count = 0
			}
		} else {
			cacheHit = true
			start = msgLen - int(arg.Count)
			if start < 0 {
				start = 0
			}
			count = int(arg.Count)
		}
	}

	if cacheHit {
		log.Debug("cacheHit.chatKey:%s.arg:%v", chatKey, *arg)

		n := 0
		for i := start; i < msgLen; i++ {
			if n >= count {
				break
			}
			n++
			reply.MsgList = append(reply.MsgList, *cd.msgList[i])
		}
		cd.msgMutex.Unlock()
		return
	}

	cd.msgMutex.Unlock()

	log.Debug("load from db")
	// not cached msg must load from db
	msgList, err = getMsgListDb(arg)
	if err != nil {
		log.Error("get msg list from db failed.err:%v", err)
		return
	}
	for i := 0; i < len(msgList); i++ {
		reply.MsgList = append(reply.MsgList, *msgList[i])
	}
	return
}

func (b *Bucket) delEmpty(chatKey string) {
	var (
		ok bool
	)
	if _, ok = b.chats[chatKey]; ok {
		delete(b.chats, chatKey)
		log.Debug("found expire chat. chatKey:%s", chatKey)
	}
}

func (b *Bucket) clean() {
	var (
		i        int
		chatKeys []string
	)
	for {
		chatKeys = b.cleaner.Clean()
		if len(chatKeys) != 0 {
			b.bLock.Lock()
			for i = 0; i < len(chatKeys); i++ {
				b.delEmpty(chatKeys[i])
			}
			b.bLock.Unlock()
			continue
		}
		time.Sleep(10 * time.Second)
	}
}
