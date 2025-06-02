package session

import (
	"fmt"
	"http2/frame"
)

//go:generate stringer -type=ErrorCode

type ErrorCode int32

const (
	ErrorCodeUnset ErrorCode = iota - 1
	ErrorCodeNoError
	ErrorCodeProtocol
	ErrorCodeInternal
	ErrorCodeFlowControl
	ErrorCodeSettingsTimeout
	ErrorCodeStreamClosed
	ErrorCodeFrameSize
	ErrorCodeRefusedStream
	ErrorCodeCancel
	ErrorCodeCompression
	ErrorCodeConnect
	ErrorCodeEnhanceYourCalm
	ErrorCodeHttp11Required
)

// Dispatcher functions return a ConnError if the client
// has triggered an http2 CONNECTION_ERROR.
type ConnError struct {
	// The Error code
	ErrorCode
	LastSid frame.Sid
	Reason  string
}

func (ce *ConnError) Error() string {
	return fmt.Sprintf("%s (last sid %d): %s", ce.ErrorCode, ce.LastSid, ce.Reason)
}
