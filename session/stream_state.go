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


func (ss StreamState) ReceivedHeader() StreamState {
	switch ss {
	case StreamStateIdle:
		return StreamStateOpen
	case StreamStateRemoteReserved:
		return StreamStateLocalClosed
	}
	panic("wrong state")
}

func (ss StreamState) SentHeader() StreamState {
	switch ss {
	case StreamStateIdle:
		return StreamStateOpen
	case StreamStateLocalReserved:
		return StreamStateRemoteClosed
	}
	panic("wrong state")
}

func (ss StreamState) ReceivedEndStream() StreamState {
	switch ss {
	case StreamStateOpen:
		return StreamStateRemoteClosed
	case StreamStateLocalClosed:
		return StreamStateClosed
	}
	panic("bad")
}

func (ss StreamState) SentEndStream() StreamState {
	switch ss {
	case StreamStateOpen:
		return StreamStateLocalClosed
	case StreamStateRemoteClosed:
		return StreamStateClosed
	}
	panic("bad")
}

func (ss StreamState) SentRstStream() StreamState {
	return StreamStateClosed
}


