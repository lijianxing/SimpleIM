package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
	log "github.com/thinkboy/log4go"
)

const (
	SUBKEY_PREFIX = "myim_route_"
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

type IRouter interface {
	Get(arg *GetRouteArg) (session *Session, err error)
	MGet(args []*GetRouteArg) (sessions []*Session, err error)
	Set(arg *SetRouteArg) (err error)
	Del(arg *DelRouteArg) (has bool, err error)
}

type RedisRouter struct {
	pool *redis.Pool
	conf *RedisConfig
}

type RedisConfig struct {
	Addr        string
	MaxIdle     int
	MaxActive   int
	IdleTimeout time.Duration
}

var (
	DefaultRouter RedisRouter
	Router        IRouter = &DefaultRouter
)

func InitRedisRouter(conf *RedisConfig) (err error) {
	return DefaultRouter.Init(conf)
}

func (rr *RedisRouter) Init(conf *RedisConfig) (err error) {
	rr.conf = conf
	rr.pool = &redis.Pool{
		MaxIdle:     conf.MaxIdle,
		MaxActive:   conf.MaxActive,
		IdleTimeout: conf.IdleTimeout,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", conf.Addr)
		},
	}
	return
}

func (rr *RedisRouter) Get(arg *GetRouteArg) (session *Session, err error) {
	if arg == nil {
		err = ErrInvalidArgument
		return
	}

	conn := rr.pool.Get()
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

func (rr *RedisRouter) MGet(args []*GetRouteArg) (res []*Session, err error) {
	// if len(args) == 0 {
	// 	return
	// }

	// conn := rr.pool.Get()
	// if conn == nil {
	// 	err = ErrInternalError
	// 	return
	// }
	// defer conn.Close()

	// var keys []string
	// for _, arg := range args {
	// 	keys = append(keys, makeKey(arg.Key))
	// }
	// if datas, err := redis.String(conn.Do("MGET", keys)); err == nil {
	// 	for _, data := range datas {
	// 	}
	// }
	return
}

func (rr *RedisRouter) Set(arg *SetRouteArg) (err error) {
	if arg == nil {
		err = ErrInvalidArgument
		return
	}

	conn := rr.pool.Get()
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

func (rr *RedisRouter) Del(arg *DelRouteArg) (has bool, err error) {
	if arg == nil {
		log.Error("del router arg is nil")
		err = ErrInvalidArgument
		return
	}

	conn := rr.pool.Get()
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
	return fmt.Sprintf("%s%s", SUBKEY_PREFIX, key)
}
