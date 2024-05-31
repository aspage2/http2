package session

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"http2/frame"
	"http2/hpack"
	"io"
)

var ClientPreface = []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
var UnexpectedPreface = errors.New("unexpected preface")

// A Session object represents an open connection
// between this server and a client.
type Session struct {
	Incoming io.Reader
	Outgoing io.Writer

	lastStream frame.Sid

	Streams map[frame.Sid]*Stream

	LookupTable *hpack.HeaderLookupTable

	Handle func(map[string][]string, []byte) (map[string][]string, []byte)
}

func NewSession(rd io.Reader, wr io.Writer) *Session {
	var sess Session
	sess.Incoming = rd
	sess.Outgoing = wr
	sess.lastStream = 0
	sess.Streams = make(map[frame.Sid]*Stream)
	sess.LookupTable = hpack.NewHeaderLookupTable()
	return &sess
}

func (sess *Session) ConsumePreface() error {
	preface := make([]byte, 24)
	n, err := io.ReadFull(sess.Incoming, preface)
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

func (sess *Session) Serve() error {
	if err := sess.ConsumePreface(); err != nil {
		return err
	}
	globalStream := sess.Stream(0)
	// Server must initiate communications by sending
	// a settings frame with initial settings. Leave it
	// empty for now until we implement actual setting
	// enforcement
	if err := globalStream.SendFrame(frame.FrameSettings, 0, nil); err != nil {
		return err
	}

	// Client will also send their initial settings.
	// receive and acknowledge the frame.
	_, data, err := sess.ExpectFrame(frame.FrameSettings, 0)
	if err != nil {
		return err
	}
	sl := SettingsListFromFramePayload(data)
	fmt.Println("---(INITIAL CLIENT SETTINGS)---")
	for _, item := range sl.Settings {
		fmt.Printf("%s = %d\n", item.Type, item.Value)
	}
	globalStream.SendFrame(frame.FrameSettings, STGS_ACK, nil)

	for {
		fh, data, err := sess.ReadFrame()
		if err != nil {
			return err
		}
		if err := sess.Dispatch(fh, data); err != nil {
			return err
		}
	}
}

func (sess *Session) readHeader() (*frame.FrameHeader, error) {
	fh := new(frame.FrameHeader)
	err := fh.Unmarshal(sess.Incoming)
	if err != nil {
		return nil, err
	}
	return fh, nil
}

func (sess *Session) readPayload(n uint32) ([]uint8, error) {
	data := make([]uint8, n)
	_, err := io.ReadFull(sess.Incoming, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Read a frame from the incoming connection. Returns the
// frame header object + the frame payload if nonzero. If
// the frame doesn't have a payload, ReadFrame returns a nil
// slice.
func (sess *Session) ReadFrame() (*frame.FrameHeader, []uint8, error) {
	fh, err := sess.readHeader()
	if err != nil {
		return nil, nil, err
	}
	if fh.Length == 0 {
		return fh, nil, nil
	}
	data, err := sess.readPayload(fh.Length)
	if err != nil {
		return nil, nil, err
	}
	return fh, data, nil
}

func (sess *Session) ExpectFrame(typ frame.FrameType, sid frame.Sid) (*frame.FrameHeader, []uint8, error) {
	fh, err := sess.readHeader()
	if err != nil {
		return nil, nil, err
	}
	if fh.Type != typ {
		return nil, nil, fmt.Errorf("expected type %s, got %s", typ, fh.Type)
	}
	if fh.Sid != sid {
		return nil, nil, fmt.Errorf("expected sid %s, got %s", sid, fh.Sid)
	}
	if fh.Length == 0 {
		return fh, nil, nil
	}
	data, err := sess.readPayload(fh.Length)
	if err != nil {
		return nil, nil, err
	}
	return fh, data, nil
}

func (sess *Session) Stream(sid frame.Sid) *Stream {
	if st, ok := sess.Streams[sid]; ok {
		return st
	}
	st := NewStream(sid, sess)
	sess.Streams[sid] = st
	return st
}

// Dispatch a frame header and payload to the appropriate handler.
func (sess *Session) Dispatch(fh *frame.FrameHeader, data []uint8) error {
	// Last stream interacted with for error-sending
	sess.lastStream = fh.Sid
	switch fh.Type {
	case frame.FrameSettings:
		if fh.Flag(0) {
			return nil
		}
		// Must acknowledge new settings frame
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

	case frame.FrameData:
		buf := bytes.NewReader(data)
		dataSize := fh.Length

		// Frame is padded. The first byte of the payload
		// is the pad length.
		if fh.Flag(3) {
			c, err := buf.ReadByte()
			if err != nil {
				return err
			}
			dataSize -= uint32(c) + 1
		}
		newData := make([]uint8, dataSize)
		_, err := buf.Read(newData)
		if err != nil {
			return err
		}
		st := sess.Stream(fh.Sid)
		st.ExtendData(newData)

		// Bit 0 is END_STREAM
		if fh.Flag(0) {
			st.SetLocalClosed()
		}

	default:
		fmt.Printf("Unknown frame\n%s\n%s\n", fh, hex.Dump(data))
	}
	st := sess.Stream(fh.Sid)
	if fh.Sid != 0 && st.IsRemoteClosed() {
		retHeaders, retData := sess.Handle(st.headers, st.data)
		hl := hpack.NewHeaderList(sess.LookupTable)
		if retHeaders == nil {
			hl.Put(":status", "201")
		} else {
			if _, ok := retHeaders[":status"]; !ok {
				if retData != nil {
					hl.Put(":status", "200")
				} else {
					hl.Put(":status", "201")
				}
			}
			for k, vs := range retHeaders {
				if vs == nil {
					continue
				}
				for _, v := range vs {
					hl.Put(k, v)
				}
			}
		}
		flg := FLAG_END_HEADERS
		if retData == nil {
			flg |= FLAG_END_STREAM
		}
		st.SendFrame(frame.FrameHeaders, flg, hl.Dump())
		if retData == nil {
			return nil
		}
		st.SendFrame(frame.FrameData, FLAG_END_STREAM, retData)
		st.SetLocalClosed()
		sess.SendGoaway(fh.Sid, StreamErrorNoError, "")
	}
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

	st := sess.Stream(fh.Sid)
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
		st.AddHeader(k, v)
		if hdr.ShouldIndex() {
			sess.LookupTable.Insert(k, v)
		}
	}
	if fh.Flag(0) {
		st.SetRemoteClosed()
	}
	return nil
}
