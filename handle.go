package main

import (
	"fmt"
	"bufio"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"time"
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

	buf := bufio.NewReader(conn)

	cont := make(chan struct{})

	go func() {
		for {
			select {
			case <-cont:
				continue
			case <-time.After(1 * time.Second):
				conn.Close()
			}
		}
	}()

	if err := ConsumePreface(buf); err != nil {
		return err
	}

	for {
		data, err := io.ReadAll(buf)
		if err != nil {
			fmt.Println(hex.Dump(data))
			return err
		}
	}
}
