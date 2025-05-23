package session

//go:generate stringer -type=StreamState

type StreamState uint8

const (
	StreamStateUnset StreamState = iota
	StreamStateIdle
	StreamStateLocalReserved
	StreamStateRemoteReserved
	StreamStateOpen
	StreamStateRemoteClosed
	StreamStateLocalClosed
	StreamStateClosed
)

func (ss StreamState) Idle() bool {
	return ss == StreamStateIdle
}

