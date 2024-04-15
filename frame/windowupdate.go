package frame

import (
	"fmt"
	"encoding/binary"
	"errors"
	"strings"
)

type WindowPayload struct {
	ReservedBit   bool
	IncrementSize uint32
}

func NewWindowPayload(rb bool, inc uint32) *WindowPayload {
	return &WindowPayload{
		ReservedBit:   rb,
		IncrementSize: inc,
	}
}
func WindowPayloadFromFrame(fr *Frame) (*WindowPayload, error) {
	if fr.Length != 4 {
		return nil, errors.New("window increment payloads must be length 4.")
	}
	var ret WindowPayload
	ret.ReservedBit = fr.Data[0]&0x80 != 0
	ret.IncrementSize = binary.BigEndian.Uint32(fr.Data)
	return &ret, nil
}

func (w *WindowPayload) ToFrame(fr *Frame) {
	fr.Length = 4
	fr.Flags = 0
	fr.Type = FrameTypeWindowUpdate
	fr.Data = make([]byte, 4)
	binary.BigEndian.PutUint32(fr.Data, w.IncrementSize)
	if w.ReservedBit {
		fr.Data[0] |= 0x80
	}
}

func (w *WindowPayload) String() string {
	var sb strings.Builder
	sb.WriteString("Type: WINDOW UPDATE\n")
	fmt.Fprintf(&sb, "Reserved bit set? %v\nIncrement Size = %d", w.ReservedBit, w.IncrementSize)
	return sb.String()
}
