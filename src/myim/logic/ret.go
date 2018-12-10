package main

type RetCode struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

var (
	OK            = RetCode{1000, "成功"}
	InvalidParams = RetCode{100001, "参数错误"}
	ServerError   = RetCode{100004, "服务异常"}

	RecordNotFound = RetCode{200001, "记录不存在"}
	RecordExist    = RetCode{200002, "记录已存在"}
)
