package session

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"http2/frame"
	"http2/hpack"
	"io"
)

type ConnError struct {
	ErrorCode
	LastSid frame.Sid
	Reason string
}

func (ce *ConnError) Error() string {
	return fmt.Sprintf("%s (last sid %d): %s", ce.ErrorCode, ce.LastSid, ce.Reason)
}

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

	Handler Handler
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

func (sess *Session) initialHandshake() error {
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
	fmt.Println("-------------------------------")
	globalStream.SendFrame(frame.FrameSettings, STGS_ACK, nil)
	return nil
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

// Continue accepting and dispatching packets on this session
// until the connection closes or an error occurs.
func (sess *Session) Serve() error {
	if err := sess.initialHandshake(); err != nil {
		fmt.Println(err)
		return err
	}
	var (
		fh *frame.FrameHeader
		data []byte
		err error
	)
	for {
		fh, data, err = sess.ReadFrame()
		if err != nil {
			break
		}
		fmt.Println(fh)
		err = sess.Dispatch(fh, data)
		if err != nil {
			break
		}
	}
	if err != nil {
		if ce, ok := err.(*ConnError); ok {
			sess.SendGoaway(ce.LastSid, ce.ErrorCode, ce.Reason)
		}
		return err
	}
	return nil
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
		return nil, nil, fmt.Errorf("expected sid %d, got %d", sid, fh.Sid)
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
	st := sess.Stream(fh.Sid)

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
		sess.SendGoaway(sess.lastStream, ErrorCodeNoError, "")
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
		if _, err := st.Body.Write(newData); err != nil {
			return err
		}

		// Bit 0 is END_STREAM
		if fh.Flag(0) {
			st.Body.Close()
			var resp Response
			resp.Body = bytes.NewBuffer(nil)
			sess.Handler.Handle(
				&Request{Headers: st.InHeaders.Headers, Body: st.Body}, 
				&resp,
			)
			hl := hpack.NewHeaderList(sess.LookupTable)
			foundStatus := false
			for _, pair := range resp.Headers {
				if pair.k == ":status" {
					foundStatus = true
				}
				hl.Put(pair.k, pair.v)
			}
			if !foundStatus {
				if resp.Body.Len() == 0 {
					hl.Put(":status", "200")
				} else {
					hl.Put(":status", "201")
				}
			}
			flg := FLAG_END_HEADERS
			if resp.Body.Len() == 0 {
				flg |= FLAG_END_STREAM
			}
			st.SendFrame(frame.FrameHeaders, flg, hl.Dump())
			data, _ := io.ReadAll(resp.Body)
			st.SendFrame(frame.FrameData, FLAG_END_STREAM, data)
			st.State = st.State.SentEndStream()
			sess.lastStream = fh.Sid
			return sess.ConnError(ErrorCodeNoError, "")
		}

	default:
		fmt.Println("(I don't know what to do with this frame)")
	}
	sess.lastStream = fh.Sid
	return nil
}

func (sess *Session) SendGoaway(lastSid frame.Sid, code ErrorCode, message string) {
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

func (sess *Session) ConnError(code ErrorCode, reason string) error {
	return &ConnError{
		ErrorCode: code,
		LastSid: sess.lastStream,
		Reason: reason,
	}
}

func (sess *Session) HandleHeader(fh *frame.FrameHeader, data []uint8) error {
	totRead := 0
	padLength := 0

	st := sess.Stream(fh.Sid)

	st.State = st.State.ReceivedHeader()

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
	tr, err := sess.ReadHeaders(st.InHeaders.Add, data, totRead, padLength)
	if err != nil {
		return err
	}
	totRead += tr
	// End Stream
	if fh.Flag(0) {
		st.State = st.State.ReceivedEndStream()
		st.Body.Close()
	}
	// End of headers
	if fh.Flag(2) {
		st.InHeaders.Closed = true
	}
	return nil
}

func (sess *Session) ReadHeaders(cb func(k, v string), data []byte, totRead int, padLength int) (int, error) {
	for totRead < len(data) - padLength {
		hdr, numRead, err := hpack.NextHeader(data[totRead:])
		if err != nil {
			return 0, err
		}
		totRead += numRead
		k, v, err := hdr.Resolve(sess.LookupTable)
		if err != nil {
			return 0, err
		}
		cb(k, v)
		if hdr.ShouldIndex() {
			sess.LookupTable.Insert(k, v)
		}
	}
	return totRead, nil
}

func (sess *Session) DoRequest(sid frame.Sid) error {
	stream := sess.Stream(sid)
	stream.InHeaders.Closed = true
	return nil
}
