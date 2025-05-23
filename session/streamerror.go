package session

//go:generate stringer -type=ErrorCode

type ErrorCode uint32

const (
	ErrorCodeNoError ErrorCode = iota
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
