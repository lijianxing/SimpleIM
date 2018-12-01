package main

type Auther interface {
	Auth(appId, userId, token string) (ok bool)
}

type DefaultAuther struct {
}

func NewDefaultAuther() *DefaultAuther {
	return &DefaultAuther{}
}

func (a *DefaultAuther) Auth(appId, userId, token string) (ok bool) {
	// var err error
	// if userId, err = strconv.ParseInt(token, 10, 64); err != nil {
	// 	userId = 0
	// 	appId = 0
	// }
	// return
	ok = true
	return
}
