package main

import "errors"

var (
	ErrorBadRequest = errors.New("bad request")
	ErrorInternal   = errors.New("internal error")
	ErrorUnknown    = errors.New("unknown error")
)
