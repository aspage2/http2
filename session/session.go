package session

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"http2/frame"
	"http2/hpack"
	"io"
)

// A Session object represents an open connection
// between this server and a client.
type Session struct {
	Incoming io.Reader
	Outgoing io.Writer

	lastStream frame.Sid

	LookupTable *hpack.HeaderLookupTable
}

func NewSession(rd io.Reader, wr io.Writer) *Session {
	var sess Session
	sess.Incoming = rd
	sess.Outgoing = wr

	sess.lastStream = 0

	sess.LookupTable = hpack.NewHeaderLookupTable()
	return &sess
}

func (sess *Session) Stream(sid frame.Sid) *Stream {
	return &Stream{Sid: sid, Session: sess}
}

// Dispatch a frame header and payload to the appropriate handler.
func (sess *Session) Dispatch(fh *frame.FrameHeader, data []uint8) error {
	fmt.Println(fh)
	sess.lastStream = fh.Sid
	switch fh.Type {
	case frame.FrameSettings:
		sl := SettingsListFromFramePayload(data)
		for _, item := range sl.Settings {
			fmt.Printf("%s = %d\n", item.Type, item.Value)
		}
		sess.Stream(0).SendFrame(frame.FrameSettings, STGS_ACK, nil)
	case frame.FrameHeaders:
		if err := sess.HandleHeader(fh, data); err != nil {
			return err
		}
	case frame.FrameGoaway:
		gf := GoawayFrameFromPayload(data)
		fmt.Println("Received a GOAWAY from client")
		fmt.Println(gf.String())
		sess.SendGoaway(sess.lastStream, StreamErrorNoError, "")
		return errors.New("client goaway")
	default:
		fmt.Println(hex.Dump(data))
	}
	fmt.Println("------------------------------------------------------------")
	return nil
}

func (sess *Session) SendGoaway(lastSid frame.Sid, code StreamError, message string) {
	var gf GoawayFrame
	gf.LastStreamId = lastSid
	gf.ErrorCode = code
	gf.DebugInfo = []byte(message)

	sess.Stream(0).SendFrame(frame.FrameGoaway, 0, gf.Marshal())
}

const (
	FLAG_END_STREAM  uint8 = 0x01
	FLAG_END_HEADERS uint8 = 0x04
	FLAG_PADDED      uint8 = 0x08
	FLAG_PRIORITY    uint8 = 0x20
)

func (sess *Session) HandleHeader(fh *frame.FrameHeader, data []uint8) error {
	totRead := 0
	padLength := 0

	// End of headers
	if !fh.Flag(2) {
		// TODO: This server doesn't support continuation frames yet. fix it!
		return errors.New("END_OF_HEADERS not set. This server doesn't support continuation")
	}

	// Padded
	if fh.Flag(3) {
		padLength = int(data[0])
		totRead += 1
	}
	// Priority
	if fh.Flag(5) {
		depSid := binary.BigEndian.Uint32(data[totRead:])
		weight := data[totRead+4]
		fmt.Printf("STREAM DEPENDENCY: %d --> %d (weight %d)\n", fh.Sid, depSid, weight)
		totRead += 5
	}
	for totRead < len(data)-padLength {
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
	// End of stream
	if fh.Flag(0) {
		fmt.Printf("END OF STREAM %d\n", fh.Sid)
	}
	fmt.Println(sess.LookupTable)
	return nil
}
