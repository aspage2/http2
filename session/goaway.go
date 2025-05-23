package session

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"http2/frame"
)

type GoawayFrame struct {
	LastStreamId frame.Sid
	ErrorCode    ErrorCode
	DebugInfo    []uint8
}

func GoawayFrameFromPayload(data []uint8) *GoawayFrame {
	var ret GoawayFrame
	ret.LastStreamId = frame.Sid(binary.BigEndian.Uint32(data) & 0x7fffffff)
	ret.ErrorCode = ErrorCode(binary.BigEndian.Uint32(data[4:]))
	ret.DebugInfo = make([]uint8, len(data)-8)
	copy(ret.DebugInfo, data[8:])
	return &ret
}

func (gf *GoawayFrame) String() string {
	return fmt.Sprintf("Last Stream: %d\nError: %s\n%s\n", gf.LastStreamId, gf.ErrorCode, hex.Dump(gf.DebugInfo))
}

func (gf *GoawayFrame) Marshal() []uint8 {
	l := 8
	if gf.DebugInfo != nil {
		l += len(gf.DebugInfo)
	}
	data := make([]uint8, 8+len(gf.DebugInfo))
	binary.BigEndian.PutUint32(data, uint32(gf.LastStreamId))
	binary.BigEndian.PutUint32(data[4:], uint32(gf.ErrorCode))
	if gf.DebugInfo != nil {
		copy(data[8:], gf.DebugInfo)
	}
	return data
}
