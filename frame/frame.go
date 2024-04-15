package frame

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type FrameType uint8

const (
	// Frame Types
	FrameTypeData         FrameType = 0x0
	FrameTypeHeaders      FrameType = 0x1
	FrameTypePriority     FrameType = 0x2
	FrameTypeResetStream  FrameType = 0x3
	FrameTypeSettings     FrameType = 0x4
	FrameTypePushPromise  FrameType = 0x5
	FrameTypePing         FrameType = 0x6
	FrameTypeGoAway       FrameType = 0x7
	FrameTypeWindowUpdate FrameType = 0x8
	FrameTypeContinuation FrameType = 0x9

	RBitMask uint32 = 0xEFFFFFFF
)

type Frame struct {
	Type   FrameType
	Flags  uint8
	Length uint32
	Sid    uint32
	Data   []byte
}

func ReadFrame(rd io.Reader, fh *Frame) error {
	buf := make([]byte, 9)
	n, err := rd.Read(buf)
	if err != nil {
		return err
	}
	if n != 9 {
		return errors.New("invalid frame header")
	}
	fh.Length = uint32(
		(uint32(buf[0]) << 16) |
			(uint32(buf[1]) << 8) |
			(uint32(buf[2])),
	)
	fh.Type = FrameType(buf[3])
	fh.Flags = buf[4]
	fh.Sid = binary.BigEndian.Uint32(buf[5:])
	fh.Data = make([]byte, fh.Length)
	_, err = rd.Read(fh.Data)
	return err
}

func (f *Frame) Marshal() []byte {
	// TODO: As a default, frame size is limited to 2^14.
	// What to do here if length > 2^14?
	l := f.Length
	var ret = make([]byte, 9+l)

	ret[0] = byte(l >> 16)
	ret[1] = byte(l >> 8)
	ret[2] = byte(l)

	ret[3] = uint8(f.Type)
	ret[4] = f.Flags
	binary.BigEndian.PutUint32(ret[5:], f.Sid&RBitMask)
	copy(ret[9:], f.Data)

	return ret
}

type FramePayload interface {
	ToFrame(*Frame)
}

func (f *Frame) GetPayload() (FramePayload, error) {
	switch f.Type {
	case FrameTypeSettings:
		return SettingsFromFrame(f)
	case FrameTypeWindowUpdate:
		return WindowPayloadFromFrame(f)
	default:
		uf := UnknownFrame(*f)
		return &uf, nil
	}
}

func (f *Frame) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "=== FRAME (SID %d) ===\n", f.Sid)
	p, err := f.GetPayload()
	if err != nil {
		fmt.Fprintf(&sb, "<error: %s>\n", err)
	} else {
		fmt.Fprintln(&sb, p)
	}
	sb.WriteString("=========")
	return sb.String()
}

type UnknownFrame Frame

func (u *UnknownFrame) ToFrame(f *Frame) {
	f.Type = u.Type
	f.Sid = u.Sid
	f.Flags = u.Flags
	f.Length = u.Length
	f.Data = make([]byte, len(u.Data))
	copy(f.Data, u.Data)
}

func (u *UnknownFrame) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Type: Unknown(0x%x)\n", u.Type)
	var flags []string
	for i := 0; i < 8; i++ {
		if u.Flags&(1<<i) != 0 {
			a := strconv.Itoa(i)
			flags = append(flags, a)
		}
	}
	sb.WriteString("Flags: ")
	sb.WriteString(strings.Join(flags, ", "))
	sb.WriteString("\nPayload:\n")
	sb.WriteString(hex.Dump(u.Data))
	return sb.String()
}
