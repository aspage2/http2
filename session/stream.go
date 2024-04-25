package session

import (
	"fmt"
	"bytes"
	"http2/frame"
	"io"
)

// A stream represents a single two-way channel
// within a session.
type Stream struct {
	Sid frame.Sid
	Session *Session
}

func (stream *Stream) SendFrame(typ frame.FrameType, flags uint8, data []uint8) error {
	fh := new(frame.FrameHeader)
	fh.Sid = stream.Sid
	fh.Flags = flags
	fh.Type = typ
	fh.Length = uint32(len(data))

	if err := fh.Marshal(stream.Session.Outgoing); err != nil {
		return err
	}
	if data == nil || len(data) == 0 {
		return nil
	}
	_, err := io.Copy(stream.Session.Outgoing, bytes.NewReader(data))
	return err
}

func (stream *Stream) ExpectFrameType(typ frame.FrameType) (*frame.FrameHeader, error) {
	fh := new(frame.FrameHeader)
	err := fh.Unmarshal(stream.Session.Incoming)
	if err != nil {
		return nil, err
	}
	if fh.Type != typ {
		return nil, fmt.Errorf("expected %s, got %s", typ, fh.Type)
	}
	if fh.Sid != stream.Sid {
		return nil, fmt.Errorf("expected stream %d, got %d", stream.Sid, fh.Sid)
	}
	return fh, nil
}

