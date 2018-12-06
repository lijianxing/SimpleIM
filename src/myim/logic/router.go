package main

import (
	"encoding/json"
	"fmt"
	"myim/libs/define"

	"github.com/garyburd/redigo/redis"
	log "github.com/thinkboy/log4go"
)

type Session struct {
	ServerId int32
	Seq      int32
}

// 用户路由 (appId+userId -> session)
type GetRouteArg struct {
	Key string
}

type SetRouteArg struct {
	Key     string
	Session Session
}

type DelRouteArg struct {
	Key string
}

func getRoute(arg *GetRouteArg) (session *Session, err error) {
	if arg == nil {
		err = ErrInvalidArgument
		return
	}

	conn := RedisManager.GetConn()
	if conn == nil {
		log.Error("get redis conn failed")
		err = ErrInternalError
		return
	}
	defer conn.Close()

	var (
		data string
	)

	if data, err = redis.String(conn.Do("GET", makeKey(arg.Key))); err == nil {
		session, err = unserializeSession(data)
	}
	return
}

func mGetRoute(args []*GetRouteArg) (res []*Session, err error) {
	if len(args) == 0 {
		return
	}

	conn := RedisManager.GetConn()
	if conn == nil {
		err = ErrInternalError
		return
	}
	defer conn.Close()

	var (
		keys    []interface{}
		session *Session
	)

	for _, arg := range args {
		keys = append(keys, makeKey(arg.Key))
	}

	if datas, err := redis.Strings(conn.Do("MGET", keys...)); err == nil {
		for _, data := range datas {
			if len(data) > 0 {
				session, err = unserializeSession(data)
				res = append(res, session)
			} else {
				res = append(res, nil)
			}
		}
	}
	return
}

func setRoute(arg *SetRouteArg) (err error) {
	if arg == nil {
		err = ErrInvalidArgument
		return
	}

	conn := RedisManager.GetConn()
	if conn == nil {
		log.Error("get redis conn failed")
		err = ErrInternalError
		return
	}
	defer conn.Close()

	session, err := serializeSession(&arg.Session)
	if err != nil {
		log.Error("serialize session failed.err=%v", err)
		return
	}

	key := makeKey(arg.Key)

	if Conf.SessionExpireSec > 0 {
		_, err = conn.Do("SET", key, session, "EX", Conf.SessionExpireSec)
	} else {
		_, err = conn.Do("SET", key, session)
	}

	if err != nil {
		log.Error("set router failed.key:%s, session:%s", key, session)
	}

	return
}

func delRoute(arg *DelRouteArg) (has bool, err error) {
	if arg == nil {
		log.Error("del router arg is nil")
		err = ErrInvalidArgument
		return
	}

	conn := RedisManager.GetConn()
	if conn == nil {
		log.Error("get redis conn failed")
		err = ErrInternalError
		return
	}
	defer conn.Close()

	has, err = redis.Bool(conn.Do("DEL", makeKey(arg.Key)))
	return
}

func serializeSession(s *Session) (res string, err error) {
	var (
		data []byte
	)

	if s == nil {
		log.Error("marshal session is nil")
		err = ErrInvalidArgument
		return
	}

	if data, err = json.Marshal(*s); err == nil {
		res = string(data)
	}
	return
}

func unserializeSession(data string) (res *Session, err error) {
	if len(data) == 0 {
		log.Error("json unmarshal session failed.data is empty")
		err = ErrInvalidArgument
	}
	var s Session
	err = json.Unmarshal([]byte(data), &s)
	if err != nil {
		log.Error("json unmarshal session failed.data:%s.err:%v", data, err)
		return
	}
	res = &s
	return
}

func makeKey(key string) string {
	return fmt.Sprintf("%s%s", define.USER_ROUTE_PREFIX, key)
}
