package frame

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// The Sid (or Stream ID) is the HTTP2 stream identifier for 
// a particular request stream
type Sid uint32

// FrameHeaders represent the 9-octet metadata header
// that heads each HTTP frame.
type FrameHeader struct {
	Length uint32
	Type   FrameType
	Sid    Sid
	Flags  uint8
}

// Flag returns whether bit `n` of
// the flags are set on the frame header.
func (fh *FrameHeader) Flag(n int) bool {
	if n < 0 || n >= 8 {
		panic("flag-access outside of the range [0, 8)")
	}
	return (fh.Flags>>n)&0x1 != 0
}

// Unmarshal parses the next 9 octets in `rd` as a FrameHeader,
// populating `fh` with the parsed data.
func (fh *FrameHeader) Unmarshal(rd io.Reader) error {
	var buf [9]uint8

	_, err := io.ReadFull(rd, buf[:])
	if err != nil {
		return err
	}
	fh.Length = uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2])
	fh.Type = FrameType(buf[3])
	fh.Flags = buf[4]
	fh.Sid = Sid(binary.BigEndian.Uint32(buf[5:]))

	return nil
}

func (fh *FrameHeader) Marshal(wr io.Writer) error {
	var buf [9]uint8

	buf[0] = uint8(fh.Length >> 16)
	buf[1] = uint8(fh.Length >> 8)
	buf[2] = uint8(fh.Length)
	buf[3] = uint8(fh.Type)
	buf[4] = uint8(fh.Flags)
	binary.BigEndian.PutUint32(buf[5:], uint32(fh.Sid))

	_, err := io.Copy(wr, bytes.NewReader(buf[:]))
	return err
}

func (fh *FrameHeader) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Frame.%s(SID %d, %d octets, ", fh.Type, fh.Sid, fh.Length)

	for i := 7; i >= 0; i-- {
		if fh.Flag(i) {
			sb.WriteRune('X')
		} else {
			sb.WriteRune('-')
		}
	}
	sb.WriteRune(')')
	return sb.String()
}
