package frame

//go:generate stringer -type=FrameType

type FrameType uint8

const (
	// IMPLICIT: Iota starts at 0
	FrameData FrameType = iota
	FrameHeaders
	FramePriority
	FrameResetStream
	FrameSettings
	FramePushPromise
	FramePing
	FrameGoaway
	FrameWindowUpdate
	FrameContinuation
)
