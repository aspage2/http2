package main

import (
	"bufio"
	"errors"
	"http2/frame"
	"http2/session"
	"io"
	"net"
)

var ClientPreface = []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
var UnexpectedPreface = errors.New("unexpected preface")

func ConsumePreface(rd io.Reader) error {
	preface := make([]byte, 24)
	n, err := rd.Read(preface)
	if err != nil {
		return err
	}
	if n != 24 {
		return UnexpectedPreface
	}
	for i, b := range ClientPreface {
		if b != preface[i] {
			return UnexpectedPreface
		}
	}
	return nil
}

func HandleConnection(conn net.Conn) error {
	defer conn.Close()
	sess := session.NewSession(conn)

	buf := bufio.NewReader(conn)
	if err := ConsumePreface(buf); err != nil {
		return err
	}

	for {
		fh := new(frame.FrameHeader)
		if err := fh.Unmarshal(buf); err != nil {
			return err
		}
		data := make([]uint8, fh.Length)
		if _, err := io.ReadFull(buf, data); err != nil {
			return err
		}
		sess.Dispatch(fh, data)
	}
}
