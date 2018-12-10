package main

import (
	"errors"
)

var (
	ErrUnknownOper     = errors.New("unknown operate")
	ErrAuthFailed      = errors.New("auth failed")
	ErrInvalidReq      = errors.New("invalid req")
	ErrDuplicatedReq   = errors.New("duplicate req")
	ErrInternalError   = errors.New("internal error")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrUnknownTarget   = errors.New("unknown target")

	ErrRouter         = errors.New("router rpc is not available")
	ErrDecodeKey      = errors.New("decode key error")
	ErrNetworkAddr    = errors.New("network addrs error, must network@address")
	ErrConnectArgs    = errors.New("connect rpc args error")
	ErrDisconnectArgs = errors.New("disconnect rpc args error")
	ErrOperateArgs    = errors.New("operate rpc args error")

	ErrGroupNotFound = errors.New("group not found")
)
