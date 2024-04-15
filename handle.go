package main

import (
	"fmt"
	"bufio"
	"errors"
	"http2/frame"
	"http2/histReader"
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

	// Instrument with a histReader to log all the bytes read
	lg := histReader.NewHistReader(conn)
	buf := bufio.NewReader(lg)

	if err := ConsumePreface(buf); err != nil {
		return err
	}

	var fh frame.Frame
	for {
		if err := frame.ReadFrame(buf, &fh); err != nil {
			return err
		}

		fmt.Println(&fh)
	}
	return nil
}
