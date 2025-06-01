package session

import (
	"bytes"
	"errors"
	"fmt"
	"http2/frame"
	"http2/hpack"
	"http2/pkg/bodystream"
	"http2/session/settings"
	"strconv"
)


type HttpCode int

const (
	CodeUnset = 0

	Ok = 200
	Created = 201
	Accepted = 202
	NonAuthoritative = 203
	NoContent = 204
	ResetContent = 205
	PartialContent = 206

	Moved = 301
	NotModified = 304
	TemporaryRedirect = 307
	PermanentRedirect = 308

	BadRequest = 400
	Unauthorized = 401
	PaymentRequired = 402
	Forbidden = 403
	NotFound = 404
	MethodNotAllowed = 405

	ServerError = 500
	NotImplemented = 501
)

type Request struct {
	Headers []stringpair
	Body *bodystream.BodyStream
}

func (req *Request) GetHeader(k string) string {
	for _, pair := range req.Headers {
		if pair.k == k {
			return pair.v
		}
	}
	return ""
}

type Response struct {
	Code HttpCode
	headersSent bool
	headers []stringpair
	body *bytes.Buffer
	stream *Stream
}

func (res *Response) SetHeader(k, v string) {
	if k == ":status" {
		code, err := strconv.Atoi(v)
		if err != nil {
			panic(fmt.Sprintf("Attempt to set non-int status code: %s", v))
		}
		res.SetResponseCode(HttpCode(code))
	} else {
		res.headers = append(res.headers, stringpair{k, v})
	}
}

func (res *Response) Flush() error {
	maxFrameSize, ok := res.stream.Context.Settings.Get(settings.MaxFrameSize)
	if !ok {
		maxFrameSize, _ = settings.Default(settings.MaxFrameSize)
	}
	buf := make([]byte, maxFrameSize)
	for res.body.Len() > 0 {
		nRead, _ := res.body.Read(buf)
		err := res.stream.SendFrame(frame.FrameData, 0, buf[:nRead])
		if err != nil {
			return err
		}
	}
	return nil
}

func (res *Response) Write(data []byte) (n int, err error) {
	if !res.headersSent {
		err := res.sendHeaders(false)
		if err != nil {
			return 0, err
		}
	}
	n, err = res.body.Write(data)
	if err != nil {
		return
	}
	maxFrameSize, ok := res.stream.Context.Settings.Get(settings.MaxFrameSize)
	if !ok {
		maxFrameSize, _ = settings.Default(settings.MaxFrameSize)
	}
	if res.body.Len() > int(maxFrameSize) {
		res.Flush()
	} 
	return
}

func (res *Response) SetResponseCode(code HttpCode) {
	res.Code = code
}

func (res *Response) sendHeaders(endStream bool) error {
	if res.headersSent {
		return errors.New("already sent headers")
	}
	hl := hpack.NewHeaderList(res.stream.Context.outgoingHeadertable)
	code := res.Code
	if code == CodeUnset {
		code = Ok
	}
	hl.Put(":status", strconv.Itoa(int(code)))
	if res.headers != nil {
		for _, pair := range res.headers {
			hl.Put(pair.k, pair.v)
		}
	}
	flags := FLAG_END_HEADERS
	if endStream {
		flags |= FLAG_END_STREAM
	}
	res.stream.SendFrame(frame.FrameHeaders, flags, hl.Dump())
	res.headersSent = true
	return nil
}

type Handler interface {
	Handle(*Request, *Response)
}

type FuncHandler func(*Request, *Response)

func (fh FuncHandler) Handle(req *Request, resp *Response) {
	fh(req, resp)
}
