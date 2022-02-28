package conn

import "errors"

var (
	ErrWrongMessageType    = errors.New("got wrong message type")
	ErrEmptyMessage        = errors.New("empty raw message")
	ErrValidate            = errors.New("error while validating incoming packet")
	ErrNotMatchAnyDistance = errors.New("does not match any request distance")
)
