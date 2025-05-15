package Bilibili

import "errors"

var (
	LoginFailed     = errors.New("login failed")
	StartLiveFailed = errors.New("start live failed")
	StopLiveFailed  = errors.New("stop live failed")
)
