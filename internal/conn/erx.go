package conn

import "errors"

var (
	ErrWrongMessageType = errors.New("got wrong message type")
	ErrEmptyMessage     = errors.New("empty raw message")
)
