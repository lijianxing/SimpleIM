package main

import (
	"database/sql"
	"myim/libs/define"
	"myim/libs/proto"
	"sort"

	log "github.com/thinkboy/log4go"
)

func saveMsgDb(arg *proto.SaveMsgArg) (msgId int64, err error) {

	var (
		sqlStr string
		stmt   *sql.Stmt
	)

	sqlStr = "insert into im_chat_msg(`app_id`, `chat_type`, `chat_code`, `from_user_id`, `msg_data`, `tag`, `create_time`) " +
		"values (?, ?, ?, ?, ?, ?, ?)"

	db := DBManager.GetDB()

	stmt, err = db.Prepare(sqlStr)
	if err != nil {
		log.Error("db prepare failed.sql:%s, err:%v", sqlStr, err)
		return 0, err
	}
	res, err := stmt.Exec(arg.AppId, arg.ChatType, arg.ChatCode, arg.FromUserId, arg.MsgData, arg.Tag, arg.CreateTime)
	if err != nil {
		log.Error("db exec failed.sql:%s, err:%v", sqlStr, err)
		return 0, err
	}

	msgId, err = res.LastInsertId()
	if err != nil {
		log.Error("db get last inserted id failed.sql:%s, err:%v", sqlStr, err)
		return 0, err
	}

	return
}

func getMsgListDb(arg *proto.GetMsgListArg) (msgList []*proto.MsgData, err error) {
	var (
		sqlStr string
		params []interface{}
		rows   *sql.Rows
	)

	sqlStr = "select `id`,`app_id`,`chat_type`,`chat_code`,`from_user_id`,`msg_data`,`tag`,`create_time` from im_chat_msg " +
		"where `app_id`=? and `chat_type`=? and `chat_code`=? "

	params = append(params, arg.AppId)
	params = append(params, arg.ChatType)
	params = append(params, arg.ChatCode)

	if arg.StartMsgId > 0 {
		if arg.Direction == define.DIRECTION_FORWARD {
			sqlStr += " and id > ? order by id"
		} else {
			sqlStr += " and id < ? order by id desc"
		}
		params = append(params, arg.StartMsgId)
	} else {
		sqlStr += " order by id desc"
	}
	sqlStr += " limit ?"
	params = append(params, arg.Count+1)

	db := DBManager.GetDB()
	if rows, err = db.Query(sqlStr, params...); err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var msgId int64
		var appId string
		var chatType int
		var chatCode string
		var fromUserId string
		var msgData string
		var tag string
		var createTime int64

		err = rows.Scan(&msgId, &appId, &chatType, &chatCode, &fromUserId, &msgData, &tag, &createTime)
		if err != nil {
			return nil, err
		}

		msgList = append(msgList, &proto.MsgData{
			MsgId:      msgId,
			PreMsgId:   0,
			FromUserId: fromUserId,
			MsgData:    msgData,
			Tag:        tag,
			CreateTime: createTime,
		})
	}

	// 按msgid 升序
	sort.Slice(msgList, func(i, j int) bool {
		return msgList[i].MsgId < msgList[j].MsgId
	})

	// PreMsgId修正
	msgLen := len(msgList)
	for i := 1; i < msgLen; i++ {
		msgList[i].PreMsgId = msgList[i-1].MsgId
	}

	if arg.StartMsgId > 0 && arg.Direction == define.DIRECTION_FORWARD {
		if msgLen > arg.Count {
			msgList = msgList[0:arg.Count]
		}
		if msgLen > 0 {
			msgList[0].PreMsgId = arg.StartMsgId // FIXME: Not Reliable
		}
	} else {
		if msgLen > arg.Count {
			msgList = msgList[msgLen-arg.Count:]
		}
	}
	return
}
