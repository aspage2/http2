package main

import (
	"bufio"
	"errors"
	"fmt"
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

	buf := bufio.NewReader(conn)
	outbuf := bufio.NewWriter(conn)
	sess := session.NewSession(buf, outbuf)

	fmt.Println("======================= NEW CONNECTION =======================")
	if err := ConsumePreface(buf); err != nil {
		return err
	}
	globalStream := sess.Stream(0)

	// Send empty settings for our preface.
	globalStream.SendFrame(frame.FrameSettings, 0, nil)
	outbuf.Flush()

	stgs, err := globalStream.ExpectFrameType(frame.FrameSettings)
	if err != nil {
		return err
	}
	data := make([]uint8, stgs.Length)
	if _, err := io.ReadFull(buf, data); err != nil {
		return err
	}
	sl := session.SettingsListFromFramePayload(data)
	fmt.Println("---(INITIAL CLIENT SETTINGS)---")
	for _, item := range sl.Settings {
		fmt.Printf("%s = %d\n", item.Type, item.Value)
	}
	fmt.Println("")
	globalStream.SendFrame(frame.FrameSettings, session.STGS_ACK, nil)
	outbuf.Flush()

	for {
		fh := new(frame.FrameHeader)
		if err := fh.Unmarshal(buf); err != nil {
			return err
		}
		data := make([]uint8, fh.Length)
		if _, err := io.ReadFull(buf, data); err != nil {
			return err
		}
		if err := sess.Dispatch(fh, data); err != nil {
			outbuf.Flush()
			return err
		}
		outbuf.Flush()
	}
}
