package main

import (
	"errors"
)

var (
	ErrLevel               = errors.New("bad level")
	ErrMatchId             = errors.New("bad match id")
	ErrOperate             = errors.New("bad operate")
	ErrAccessToken         = errors.New("bad access token")
	ErrRound               = errors.New("bad round")
	ErrAccountStatus       = errors.New("bad account status")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrResponseCodeNotOK   = errors.New("response code not ok")
)
