package session

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
