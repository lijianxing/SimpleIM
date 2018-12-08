package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"myim/libs/define"

	"github.com/garyburd/redigo/redis"
	log "github.com/thinkboy/log4go"
)

type GetGroupUserArg struct {
	AppId     string
	GroupCode string
}

type GroupUser struct {
	UserId string
}

const (
	GroupUserExpireSec = 10 * 60
)

func getGroupUser(arg *GetGroupUserArg) (groupUsers []GroupUser, err error) {
	if arg == nil {
		err = ErrInvalidArgument
		return
	}

	var (
		sqlStr string
		rows   *sql.Rows
		gusers []GroupUser
	)

	// get by cache first
	if gusers, err = getCacheGroupUser(arg); err == nil {
		log.Debug("get group user by cache. appId:%s, group:%s", arg.AppId, arg.GroupCode)
		return gusers, nil
	}

	sqlStr = "select igu.user_id as user_id from im_group ig, im_group_user igu where igu.group_id = ig.id" +
		" and ig.app_id=? and ig.group_code=?"

	db := DBManager.GetDB()
	if rows, err = db.Query(sqlStr, arg.AppId, arg.GroupCode); err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var userId string
		rows.Scan(&userId)

		groupUsers = append(groupUsers, GroupUser{
			UserId: userId,
		})
	}

	// set cache
	setCacheGroupUser(arg, groupUsers)

	return
}

func getCacheGroupUser(arg *GetGroupUserArg) (groupUsers []GroupUser, err error) {
	key := fmt.Sprintf("%s%s_%s", define.GROUP_MEMBERS_PREFIX, arg.AppId, arg.GroupCode)

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

	if data, err = redis.String(conn.Do("GET", key)); err == nil {
		groupUsers, err = unserializeGroupUsers(data)
	}
	return
}

func setCacheGroupUser(arg *GetGroupUserArg, groupUser []GroupUser) (err error) {
	key := fmt.Sprintf("%s%s_%s", define.GROUP_MEMBERS_PREFIX, arg.AppId, arg.GroupCode)

	conn := RedisManager.GetConn()
	if conn == nil {
		log.Error("get redis conn failed")
		err = ErrInternalError
		return
	}
	defer conn.Close()

	data, err := serializeGroupUsers(groupUser)
	if err != nil {
		log.Error("serialize group user failed.err=%v", err)
		return
	}

	_, err = conn.Do("SET", key, data, "EX", GroupUserExpireSec)
	if err != nil {
		log.Error("set group user cache failed.key:%s, data:%s", key, data)
	}

	return
}

func serializeGroupUsers(groupUsers []GroupUser) (res string, err error) {
	var (
		data []byte
	)

	if groupUsers == nil {
		log.Error("marshal group users is nil")
		err = ErrInvalidArgument
		return
	}

	if data, err = json.Marshal(groupUsers); err == nil {
		res = string(data)
	}
	return
}

func unserializeGroupUsers(data string) (groupUsers []GroupUser, err error) {
	if len(data) == 0 {
		log.Error("json unmarshal session failed.data is empty")
		err = ErrInvalidArgument
	}
	var guser []GroupUser
	err = json.Unmarshal([]byte(data), &guser)
	if err != nil {
		log.Error("json unmarshal group users failed.data:%s.err:%v", data, err)
		return
	}
	return guser, nil
}
