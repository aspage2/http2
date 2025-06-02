package session

import (
	"bytes"
	"http2/frame"
	"http2/pkg/bodystream"
)

type Headers struct {
	Headers []stringpair
	Closed  bool
}

func (h *Headers) Add(k, v string) {
	h.Headers = append(h.Headers, stringpair{k, v})
}

type stringpair struct {
	k string
	v string
}

// A stream represents a single two-way channel
// within a session.
type Stream struct {
	Sid frame.Sid

	Context *ConnectionContext

	State StreamState

	InHeaders *Headers
	Body      *bodystream.BodyStream
}

func NewStream(sid frame.Sid, ctx *ConnectionContext) *Stream {
	var s Stream
	s.Sid = sid
	s.Context = ctx
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

	return stream.Context.SendFrame(fh, data)
}

func (stream *Stream) Serve(ctx *ConnectionContext) {
	req := &Request{
		Body:    stream.Body,
		Headers: stream.InHeaders.Headers,
	}
	resp := &Response{
		body:   bytes.NewBuffer(nil),
		stream: stream,
	}
	ctx.Handler.Handle(req, resp)
	l := resp.body.Len()
	if !resp.headersSent {
		resp.sendHeaders(l <= 0)
	}
	if l > 0 {
		resp.Flush()
	} else {
		resp.stream.SendFrame(frame.FrameData, FLAG_END_STREAM, nil)
	}
}
