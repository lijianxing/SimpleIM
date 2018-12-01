package main

import (
	"errors"
)

var (
	ErrUnknownOper     = errors.New("unknown operate")
	ErrAuthFailed      = errors.New("auth failed")
	ErrInternalError   = errors.New("internal error")
	ErrInvalidArgument = errors.New("invalid argument")

	ErrRouter         = errors.New("router rpc is not available")
	ErrDecodeKey      = errors.New("decode key error")
	ErrNetworkAddr    = errors.New("network addrs error, must network@address")
	ErrConnectArgs    = errors.New("connect rpc args error")
	ErrDisconnectArgs = errors.New("disconnect rpc args error")
	ErrOperatetArgs   = errors.New("operate rpc args error")
)
