package frame

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var ClientPreface = []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
var UnexpectedPreface = errors.New("unexpected preface")

type Framer struct {
	Incoming io.Reader
}

func NewFramer(rd io.Reader) *Framer {
	return &Framer{
		Incoming: rd,
	}
}

func (this *Framer) readHeader() (*FrameHeader, error) {
	fh := new(FrameHeader)
	err := fh.Unmarshal(this.Incoming)
	if err != nil {
		return nil, err
	}
	return fh, nil
}

func (this *Framer) readPayload(n uint32) ([]uint8, error) {
	data := make([]uint8, n)
	_, err := io.ReadFull(this.Incoming, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Read a frame from the incoming connection. Returns the
// frame header object + the frame payload if nonzero. If
// the frame doesn't have a payload, ReadFrame returns a nil
// slice.
func (this *Framer) ReadFrame() (*Frame, error) {
	fh, err := this.readHeader()
	if err != nil {
		return nil, err
	}
	fmt.Printf("\x1b[33mReceive Frame\x1b[0m %s\n", fh)
	fr := Frame{
		FrameHeader: fh,
	}
	if fh.Length == 0 {
		return &fr, nil
	}
	data, err := this.readPayload(fh.Length)
	if err != nil {
		return nil, err
	}
	fmt.Println(hex.Dump(data[:min(len(data), 1024)]))
	fr.Data = data
	return &fr, nil
}

func (this *Framer) ConsumePreface() error {
	preface := make([]byte, 24)
	n, err := io.ReadFull(this.Incoming, preface)
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
