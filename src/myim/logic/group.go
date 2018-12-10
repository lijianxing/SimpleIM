package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"myim/libs/define"
	"myim/libs/util"

	"github.com/garyburd/redigo/redis"
	log "github.com/thinkboy/log4go"
)

type GroupInfo struct {
	GroupId    int64
	AppId      string
	GroupCode  string
	GroupName  string
	CreateTime int64
}

type CreateGroupArg struct {
	AppId     string
	GroupCode string
	GroupName string
}

type GetGroupArg struct {
	AppId     string
	GroupCode string
}

type DelGroupArg struct {
	AppId     string
	GroupCode string
}

type GetGroupUsersArg struct {
	AppId     string
	GroupCode string
}

type AddGroupUsersArg struct {
	AppId     string
	GroupCode string
	UserIds   []string
}

type RemoveGroupUsersArg struct {
	AppId     string
	GroupCode string
	UserIds   []string
}

const (
	GroupUserExpireSec = 10 * 60
)

func createGroup(arg *CreateGroupArg) error {
	var (
		sqlStr string
		stmt   *sql.Stmt
		err    error
	)

	sqlStr = "insert into im_group(`app_id`, `group_code`, `group_name`, `create_time`) " +
		" values (?, ?, ?, ?)"

	db := DBManager.GetDB()

	stmt, err = db.Prepare(sqlStr)
	if err != nil {
		log.Error("db prepare failed.sql:%s, err:%v", sqlStr, err)
		return err
	}
	_, err = stmt.Exec(arg.AppId, arg.GroupCode, arg.GroupName, util.GetTimestampMillSec())
	if err != nil {
		log.Error("db exec failed.sql:%s, err:%v", sqlStr, err)
		return err
	}

	return nil
}

func getGroup(arg *GetGroupArg) (*GroupInfo, error) {
	var (
		sqlStr string
		rows   *sql.Rows
		err    error
	)

	sqlStr = "select `id`, `group_name`, `create_time` from im_group where `app_id` = ? and `group_code` = ? limit 1"

	db := DBManager.GetDB()
	if rows, err = db.Query(sqlStr, arg.AppId, arg.GroupCode); err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var groupId int64
		var groupName string
		var createTime int64
		rows.Scan(&groupId, &groupName, &createTime)
		return &GroupInfo{GroupId: groupId, AppId: arg.AppId, GroupCode: arg.GroupCode,
			GroupName: groupName, CreateTime: createTime}, nil
	}
	return nil, nil
}

func getGroupUsers(arg *GetGroupUsersArg) (groupUsers []string, err error) {
	if arg == nil {
		err = ErrInvalidArgument
		return
	}

	var (
		sqlStr string
		rows   *sql.Rows
		gusers []string
	)

	// get by cache first
	if gusers, err = getCacheGroupUsers(arg); err == nil {
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

		groupUsers = append(groupUsers, userId)
	}

	// set cache
	setCacheGroupUsers(arg, groupUsers)

	return
}

func getCacheGroupUsers(arg *GetGroupUsersArg) (groupUsers []string, err error) {
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

func setCacheGroupUsers(arg *GetGroupUsersArg, groupUser []string) (err error) {
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

func delCacheGroupUsers(arg *DelGroupArg) (err error) {
	key := fmt.Sprintf("%s%s_%s", define.GROUP_MEMBERS_PREFIX, arg.AppId, arg.GroupCode)

	conn := RedisManager.GetConn()
	if conn == nil {
		log.Error("get redis conn failed")
		err = ErrInternalError
		return
	}
	defer conn.Close()

	_, err = conn.Do("DEL", key)
	if err != nil {
		log.Error("del group user cache failed.key:%s", key)
	}

	return
}

func serializeGroupUsers(groupUsers []string) (res string, err error) {
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

func unserializeGroupUsers(data string) (groupUsers []string, err error) {
	if len(data) == 0 {
		log.Error("json unmarshal session failed.data is empty")
		err = ErrInvalidArgument
	}
	var guser []string
	err = json.Unmarshal([]byte(data), &guser)
	if err != nil {
		log.Error("json unmarshal group users failed.data:%s.err:%v", data, err)
		return
	}
	return guser, nil
}

func addGroupUsers(arg *AddGroupUsersArg) error {
	var (
		sqlStr    string
		err       error
		groupInfo *GroupInfo
	)

	groupInfo, err = getGroup(&GetGroupArg{
		AppId:     arg.AppId,
		GroupCode: arg.GroupCode,
	})
	if err != nil {
		log.Error("check group exist failed.err:%v", err)
		return err
	}
	if groupInfo == nil {
		log.Error("group not exist.arg:%v", arg)
		return ErrGroupNotFound
	}

	now := util.GetTimestampMillSec()

	db := DBManager.GetDB()

	sqlStr = "insert ignore into im_group_user(`group_id`, `user_id`, `create_time`) " +
		" values (?, ?, ?)"

	// 事务
	tx, err := db.Begin()
	if err != nil {
		log.Error("open db tx failed.err:%v", err)
		return err
	}
	defer util.ClearTransaction(tx)

	for _, userId := range arg.UserIds {
		_, err = tx.Exec(sqlStr, groupInfo.GroupId, userId, now)
		if err != nil {
			log.Error("exec sql failed.sql:%s, groupId:%d, userId:%s, err:%v",
				sqlStr, groupInfo.GroupId, userId, err)
			return err
		}
	}

	// 事务提交
	if err := tx.Commit(); err != nil {
		log.Error("commit tx failed.arg:%v, err:%v", arg, err)
		return err
	}

	// 清除缓存
	delCacheGroupUsers(&DelGroupArg{
		AppId:     arg.AppId,
		GroupCode: arg.GroupCode,
	})

	return nil
}

func removeGroupUsers(arg *RemoveGroupUsersArg) error {
	var (
		sqlStr    string
		err       error
		groupInfo *GroupInfo
		params    []interface{}
	)

	// nothing to do
	if len(arg.UserIds) == 0 {
		return nil
	}

	groupInfo, err = getGroup(&GetGroupArg{
		AppId:     arg.AppId,
		GroupCode: arg.GroupCode,
	})
	if err != nil {
		log.Error("check group exist failed.err:%v", err)
		return err
	}
	if groupInfo == nil {
		log.Error("group not exist.arg:%v", arg)
		return ErrGroupNotFound
	}

	db := DBManager.GetDB()

	sqlStr = "delete from im_group_user where `group_id`=?"
	params = append(params, groupInfo.GroupId)

	sqlStr += " and ("
	for i, userId := range arg.UserIds {
		if i > 0 {
			sqlStr += " or "
		}
		sqlStr += " user_id = ?"
		params = append(params, userId)
	}
	sqlStr += ")"

	_, err = db.Exec(sqlStr, params...)
	if err != nil {
		log.Error("remove group user failed.sql:%s, err:%v", sqlStr, err)
		return err
	}

	// 清除缓存
	delCacheGroupUsers(&DelGroupArg{
		AppId:     arg.AppId,
		GroupCode: arg.GroupCode,
	})

	return nil
}
