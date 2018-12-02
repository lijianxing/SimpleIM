package main

import (
	"myim/libs/define"
	"myim/libs/proto"
	"time"

	log "github.com/thinkboy/log4go"
)

type Operator interface {
	// Operate process the common operation such as send message etc.
	Operate(string, *proto.Proto) error

	// Connect used for auth user and return a subkey(连接标识), hearbeat.
	Connect(*proto.Proto) (string, time.Duration, error)

	// Disconnect used for revoke the subkey.
	Disconnect(string) error
}

type DefaultOperator struct {
}

func (operator *DefaultOperator) Operate(key string, p *proto.Proto) (err error) {
	var (
		body []byte
	)
	// p是请求, 也是响应
	if p.Operation == define.OP_TEST {
		log.Debug("test operation: %s", body)
		p.Operation = define.OP_TEST_REPLY
		p.Body = []byte("{\"test\":\"come on\"}")
	} else {
		// 交给logic处理
		if err = operate(key, p); err != nil {
			return
		}
	}
	return nil
}

func (operator *DefaultOperator) Connect(p *proto.Proto) (key string, heartbeat time.Duration, err error) {
	key, heartbeat, err = connect(p)
	return
}

func (operator *DefaultOperator) Disconnect(key string) (err error) {
	var has bool
	if has, err = disconnect(key); err != nil {
		return
	}
	if !has {
		log.Warn("disconnect key: \"%s\" not exists", key)
	}
	return
}
