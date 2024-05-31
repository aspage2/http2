package session

import (
	"bytes"
	"http2/frame"
	"io"
)

// A stream represents a single two-way channel
// within a session.
type Stream struct {
	Sid     frame.Sid
	Session *Session

	headers map[string][]string
	data    []uint8

	localClosed  bool
	remoteClosed bool
}

func NewStream(sid frame.Sid, sess *Session) *Stream {
	return &Stream{
		Sid:     sid,
		Session: sess,
		headers: make(map[string][]string),
	}
}


func (st *Stream) FullyClosed() bool {
	return st.localClosed && st.remoteClosed
}

func (st *Stream) SetLocalClosed() {
	st.localClosed = true
}

func (st *Stream) IsLocalClosed() bool {
	return st.localClosed
}

func (st *Stream) IsRemoteClosed() bool {
	return st.remoteClosed
}

func (st *Stream) SetRemoteClosed() {
	st.remoteClosed = true
}

func (st *Stream) AddHeader(k, v string) {
	_, ok := st.headers[k]
	if !ok {
		st.headers[k] = []string{v}
	} else {
		st.headers[k] = append(st.headers[k], v)
	}
}

func (st *Stream) ExtendData(data []uint8) {
	newData := make([]uint8, len(data) + len(st.data))
	copy(newData, st.data)
	copy(newData[len(st.data):], data)
	st.data = newData
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

