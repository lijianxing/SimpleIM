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
	ok = true
	return
}
