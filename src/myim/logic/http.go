package main

import (
	"encoding/json"
	"io/ioutil"
	inet "myim/libs/net"
	"net"
	"net/http"
	"time"

	log "github.com/thinkboy/log4go"
)

type CreateGroupReq struct {
	AppId     string `json:"appId"`
	GroupCode string `json:"groupCode"`
	GroupName string `json:"groupName"`
}

type AddGroupUsersReq struct {
	AppId     string   `json:"appId"`
	GroupCode string   `json:"groupCode"`
	UserIds   []string `json:"userIds"`
}

type RemoveGroupUsersReq struct {
	AppId     string   `json:"appId"`
	GroupCode string   `json:"groupCode"`
	UserIds   []string `json:"userIds"`
}

type GetGroupUsersReq struct {
	AppId     string `json:"appId"`
	GroupCode string `json:"groupCode"`
}

func InitHTTP() (err error) {
	// http listen
	var network, addr string
	for i := 0; i < len(Conf.HTTPAddrs); i++ {
		httpServeMux := http.NewServeMux()
		httpServeMux.HandleFunc("/group/create", CreateGroup)
		httpServeMux.HandleFunc("/group/user/list", GetGroupUsers)
		httpServeMux.HandleFunc("/group/user/add", AddGroupUsers)
		httpServeMux.HandleFunc("/group/user/remove", RemoveGroupUsers)
		log.Info("start http listen:\"%s\"", Conf.HTTPAddrs[i])
		if network, addr, err = inet.ParseNetwork(Conf.HTTPAddrs[i]); err != nil {
			log.Error("inet.ParseNetwork() error(%v)", err)
			return
		}
		go httpListen(httpServeMux, network, addr)
	}
	return
}

func httpListen(mux *http.ServeMux, network, addr string) {
	httpServer := &http.Server{Handler: mux, ReadTimeout: Conf.HTTPReadTimeout, WriteTimeout: Conf.HTTPWriteTimeout}
	httpServer.SetKeepAlivesEnabled(true)
	l, err := net.Listen(network, addr)
	if err != nil {
		log.Error("net.Listen(\"%s\", \"%s\") error(%v)", network, addr, err)
		panic(err)
	}
	if err := httpServer.Serve(l); err != nil {
		log.Error("server.Serve() error(%v)", err)
		panic(err)
	}
}

// retWrite marshal the result and write to client(get).
func retWrite(w http.ResponseWriter, r *http.Request, body []byte, rc *RetCode, res *interface{}, start time.Time) {
	resMap := map[string]interface{}{}
	resMap["code"] = rc.Code
	resMap["msg"] = rc.Msg
	if res != nil && *res != nil {
		resMap["data"] = res
	}
	data, err := json.Marshal(resMap)
	if err != nil {
		log.Error("json.Marshal(\"%v\") error(%v)", res, err)
		return
	}
	dataStr := string(data)
	if _, err := w.Write([]byte(dataStr)); err != nil {
		log.Error("w.Write(\"%s\") error(%v)", dataStr, err)
	}
	log.Info("req: \"%s\", body:%s, get: res:\"%s\", ip:\"%s\", time:\"%fs\"",
		r.URL.String(), string(body), dataStr, r.RemoteAddr, time.Now().Sub(start).Seconds())
}

func CreateGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	var (
		bodyBytes []byte
		err       error
		res       interface{}
		rc        RetCode
		groupInfo *GroupInfo
	)

	if bodyBytes, err = ioutil.ReadAll(r.Body); err != nil {
		log.Error("ioutil.ReadAll() failed (%s)", err)
		return
	}
	defer retWrite(w, r, bodyBytes, &rc, &res, time.Now())

	var req CreateGroupReq
	err = json.Unmarshal(bodyBytes, &req)
	if err != nil {
		log.Error("parse body failed.body:%s", string(bodyBytes))
		rc = InvalidParams
		return
	}

	if req.AppId == "" || req.GroupCode == "" {
		log.Error("create group req missing appid or groupcode.", string(bodyBytes))
		rc = InvalidParams
		return
	}

	groupInfo, err = getGroup(&GetGroupArg{
		AppId:     req.AppId,
		GroupCode: req.GroupCode,
	})
	if err != nil {
		log.Error("check group exist failed.req:%v, err:%v", req, err)
		rc = ServerError
		return
	}
	if groupInfo != nil {
		log.Error("create group exist.req:%v", req)
		rc = RecordExist
		return
	}

	err = createGroup(&CreateGroupArg{
		AppId:     req.AppId,
		GroupCode: req.GroupCode,
		GroupName: req.GroupName,
	})
	if err != nil {
		log.Error("create group failed.req:%v, err:%v", req, err)
		rc = ServerError
		return
	}

	rc = OK
	return
}

func AddGroupUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	var (
		bodyBytes []byte
		err       error
		res       interface{}
		rc        RetCode
	)

	if bodyBytes, err = ioutil.ReadAll(r.Body); err != nil {
		log.Error("ioutil.ReadAll() failed (%s)", err)
		return
	}
	defer retWrite(w, r, bodyBytes, &rc, &res, time.Now())

	var req AddGroupUsersReq
	err = json.Unmarshal(bodyBytes, &req)
	if err != nil {
		log.Error("parse body failed.body:%s", string(bodyBytes))
		rc = InvalidParams
		return
	}

	if req.AppId == "" || req.GroupCode == "" || len(req.UserIds) == 0 {
		log.Error("add group users req missing appid, groupcode or userids.", string(bodyBytes))
		rc = InvalidParams
		return
	}

	err = addGroupUsers(&AddGroupUsersArg{
		AppId:     req.AppId,
		GroupCode: req.GroupCode,
		UserIds:   req.UserIds,
	})
	if err != nil {
		log.Error("add group users failed.req:%v, err:%v", req, err)
		rc = ServerError
		return
	}

	rc = OK
	return
}

func RemoveGroupUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	var (
		bodyBytes []byte
		err       error
		res       interface{}
		rc        RetCode
	)

	if bodyBytes, err = ioutil.ReadAll(r.Body); err != nil {
		log.Error("ioutil.ReadAll() failed (%s)", err)
		return
	}
	defer retWrite(w, r, bodyBytes, &rc, &res, time.Now())

	var req RemoveGroupUsersReq
	err = json.Unmarshal(bodyBytes, &req)
	if err != nil {
		log.Error("parse body failed.body:%s", string(bodyBytes))
		rc = InvalidParams
		return
	}

	if req.AppId == "" || req.GroupCode == "" || len(req.UserIds) == 0 {
		log.Error("remove group req missing appid, groupcode or userids.", string(bodyBytes))
		rc = InvalidParams
		return
	}

	err = removeGroupUsers(&RemoveGroupUsersArg{
		AppId:     req.AppId,
		GroupCode: req.GroupCode,
		UserIds:   req.UserIds,
	})
	if err != nil {
		log.Error("remove group users failed.req:%v, err:%v", req, err)
		rc = ServerError
		return
	}

	rc = OK
	return
}

func GetGroupUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	var (
		bodyBytes  []byte
		err        error
		res        interface{}
		rc         RetCode
		groupUsers []string
	)

	if bodyBytes, err = ioutil.ReadAll(r.Body); err != nil {
		log.Error("ioutil.ReadAll() failed (%s)", err)
		return
	}
	defer retWrite(w, r, bodyBytes, &rc, &res, time.Now())

	var req GetGroupUsersReq
	err = json.Unmarshal(bodyBytes, &req)
	if err != nil {
		log.Error("parse body failed.body:%s", string(bodyBytes))
		rc = InvalidParams
		return
	}

	if req.AppId == "" || req.GroupCode == "" {
		log.Error("get group users req missing appid, groupcode.", string(bodyBytes))
		rc = InvalidParams
		return
	}

	groupUsers, err = getGroupUsers(&GetGroupUsersArg{
		AppId:     req.AppId,
		GroupCode: req.GroupCode,
	})
	if err != nil {
		log.Error("add group users failed.req:%v, err:%v", req, err)
		rc = ServerError
		return
	}
	res = groupUsers

	rc = OK
	return
}
