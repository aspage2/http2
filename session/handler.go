package session

import (
	"bytes"
	"http2/pkg/bodystream"
	"strconv"
)


type Request struct {
	Headers []stringpair
	Body *bodystream.BodyStream
}

type Response struct {
	Headers []stringpair
	Body *bytes.Buffer
}

func (res *Response) SetHeader(k, v string) {
	res.Headers = append(res.Headers, stringpair{k, v})
}

func (res *Response) SetResponseCode(code int) {
	res.SetHeader(":status", strconv.Itoa(code))
}

type Handler interface {
	Handle(*Request, *Response)
}

type FuncHandler func(*Request, *Response)

func (fh FuncHandler) Handle(req *Request, resp *Response) {
	fh(req, resp)
}
