package session

import (
	"bytes"
	"http2/frame"
	"http2/pkg/bodystream"
	"io"
)

type Headers struct {
	Headers []stringpair
	Closed bool
}

func (h *Headers) Add(k, v string) {
	h.Headers = append(h.Headers, stringpair{k, v})
}

type stringpair struct { k string; v string }

// A stream represents a single two-way channel
// within a session.
type Stream struct {
	Sid     frame.Sid
	Session *Session

	State StreamState

	InHeaders *Headers
	Body *bodystream.BodyStream
}


func NewStream(sid frame.Sid, sess *Session) *Stream {
	var s Stream
	s.Sid = sid
	s.Session = sess
	s.State = StreamStateIdle
	s.InHeaders = new(Headers)
	s.Body = bodystream.NewBodyStream()
	return &s
}

// Send a frame to the client.
// If the body is nil, it is treated as a zero-length payload.
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
