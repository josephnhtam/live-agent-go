package transport

import "errors"

var (
	ErrSessionClosed   = errors.New("session closed")
	ErrMessageNotReady = errors.New("message channel not ready")
)
