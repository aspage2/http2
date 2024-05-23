package session

//go:generate stringer -type=StreamError

type StreamError uint32

const (
	StreamErrorNoError StreamError = iota
	StreamErrorProtocol
	StreamErrorInternal
	StreamErrorFlowControl
	StreamErrorSettingsTimeout
	StreamErrorStreamClosed
	StreamErrorFrameSize
	StreamErrorRefusedStream
	StreamErrorCancel
	StreamErrorCompression
	StreamErrorConnect
	StreamErrorEnhanceYourCalm
	StreamErrorHttp11Required
)
