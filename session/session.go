package session

import (
	"encoding/hex"
	"fmt"
	"http2/frame"
	"http2/hpack"
	"io"
)

// A Session object represents an open connection
// between this server and a client.
type Session struct {
	Incoming io.Reader

	LookupTable *hpack.HeaderLookupTable
}

func NewSession(rd io.Reader) *Session {
	var sess Session
	sess.Incoming = rd
	sess.LookupTable = hpack.NewHeaderLookupTable()
	return &sess
}

// Dispatch a frame header and payload to the appropriate handler.
func (sess *Session) Dispatch(fh *frame.FrameHeader, data []uint8) error {
	fmt.Println(fh)
	switch fh.Type {
	case frame.FrameSettings:
		sl := SettingsListFromFramePayload(data)
		for _, item := range sl.Settings {
			fmt.Printf("%s = %d\n", item.Type, item.Value)
		}
	case frame.FrameHeaders:
		totRead := 0
		for totRead < len(data) {
			hdr, numRead, err := hpack.NextHeader(data[totRead:])
			if err != nil {
				return err
			}
			totRead += numRead
			k, v, err := hdr.Resolve(sess.LookupTable)
			if err != nil {
				return err
			}
			fmt.Printf("%s: %s\n", k, v)
			if hdr.ShouldIndex() {
				sess.LookupTable.Insert(k, v)
			}
		}
		fmt.Println(sess.LookupTable)
	default:
		fmt.Println(hex.Dump(data))
	}
	fmt.Println("------------------------------------------------------------")
	return nil
}
