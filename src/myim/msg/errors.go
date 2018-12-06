package main

import (
	"errors"
)

var (
	ErrUnknownOper     = errors.New("unknown operate")
	ErrInvalidReq      = errors.New("invalid req")
	ErrDuplicatedReq   = errors.New("duplicate req")
	ErrInternalError   = errors.New("internal error")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrUnknownTarget   = errors.New("unknown target")

	ErrDecodeKey   = errors.New("decode key error")
	ErrNetworkAddr = errors.New("network addrs error, must network@address")
)
