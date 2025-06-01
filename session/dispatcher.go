package session

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"http2/frame"
	"http2/hpack"
	"http2/session/settings"
	"io"
)

var ClientPreface = []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
var UnexpectedPreface = errors.New("unexpected preface")

// A Dispatcher object represents an open connection
// between this server and a client.
type Dispatcher struct {
	Ctx *ConnectionContext
	lastStream frame.Sid
	Streams map[frame.Sid]*Stream
}

func NewDispatcher(ctx *ConnectionContext) *Dispatcher {
	var sess Dispatcher
	sess.Ctx = ctx
	sess.lastStream = 0
	sess.Streams = make(map[frame.Sid]*Stream)
	return &sess
}

func (sess *Dispatcher) initialHandshake() error {
	err := ConsumePreface(sess.Ctx.incoming)
	if err != nil {
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
	sl := settings.SettingsListFromFramePayload(data)
	fmt.Println("---(INITIAL CLIENT SETTINGS)---")
	fmt.Print(sl)
	fmt.Println("-------------------------------")
	globalStream.SendFrame(frame.FrameSettings, settings.STGS_ACK, nil)
	return nil
}

func ConsumePreface(rd io.Reader) error {
	preface := make([]byte, 24)
	n, err := io.ReadFull(rd, preface)
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
func (sess *Dispatcher) Serve() error {
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
		err = sess.Dispatch(fh, data)
		if err != nil {
			break
		}
	}
	if err != nil {
		if ce, ok := err.(*ConnError); ok {
			fmt.Printf("\x1b[32mERROR ERROR\x1b[0m %s\n", ce)
			sess.SendGoaway(ce.LastSid, ce.ErrorCode, ce.Reason)
		}
		return err
	}
	return nil
}

func (sess *Dispatcher) readHeader() (*frame.FrameHeader, error) {
	fh := new(frame.FrameHeader)
	err := fh.Unmarshal(sess.Ctx.incoming)
	if err != nil {
		return nil, err
	}
	return fh, nil
}

func (sess *Dispatcher) readPayload(n uint32) ([]uint8, error) {
	data := make([]uint8, n)
	_, err := io.ReadFull(sess.Ctx.incoming, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Read a frame from the incoming connection. Returns the
// frame header object + the frame payload if nonzero. If
// the frame doesn't have a payload, ReadFrame returns a nil
// slice.
func (sess *Dispatcher) ReadFrame() (*frame.FrameHeader, []uint8, error) {
	fh, err := sess.readHeader()
	if err != nil {
		return nil, nil, err
	}
	fmt.Printf("\x1b[33mReceive Frame\x1b[0m %s\n", fh)
	if fh.Length == 0 {
		return fh, nil, nil
	}
	data, err := sess.readPayload(fh.Length)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println(hex.Dump(data[:min(len(data), 1024)]))
	return fh, data, nil
}

func (sess *Dispatcher) ExpectFrame(typ frame.FrameType, sid frame.Sid) (*frame.FrameHeader, []uint8, error) {
	fh, err := sess.readHeader()
	if err != nil {
		return nil, nil, err
	}
	fmt.Printf("\x1b[33mReceive Frame\x1b[0m %s\n", fh)
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
	fmt.Println(hex.Dump(data[:min(len(data), 1024)]))
	return fh, data, nil
}

func (sess *Dispatcher) Stream(sid frame.Sid) *Stream {
	if st, ok := sess.Streams[sid]; ok {
		return st
	}
	st := NewStream(sid, sess.Ctx)
	sess.Streams[sid] = st
	return st
}

// Dispatch a frame header and payload to the appropriate handler.
func (sess *Dispatcher) Dispatch(fh *frame.FrameHeader, data []uint8) error {
	// Last stream interacted with for error-sending
	st := sess.Stream(fh.Sid)

	switch fh.Type {
	case frame.FrameSettings:
		if fh.Flag(0) {
			return nil
		}
		// Must acknowledge new settings frame
		sess.Stream(0).SendFrame(frame.FrameSettings, settings.STGS_ACK, nil)

	case frame.FrameHeaders:
		if err := sess.HandleHeader(fh, data); err != nil {
			return err
		}

	case frame.FrameGoaway:
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
			sess.lastStream = fh.Sid
			return nil
		}

	default:
		fmt.Println("(I don't know what to do with this frame)")
	}
	sess.lastStream = fh.Sid
	return nil
}

func (sess *Dispatcher) SendGoaway(lastSid frame.Sid, code ErrorCode, message string) {
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

func (sess *Dispatcher) ConnError(code ErrorCode, reason string) error {
	return &ConnError{
		ErrorCode: code,
		LastSid: sess.lastStream,
		Reason: reason,
	}
}

func (sess *Dispatcher) HandleHeader(fh *frame.FrameHeader, data []uint8) error {
	totRead := 0
	padLength := 0

	st := sess.Stream(fh.Sid)

	st.State = st.State.ReceivedHeader()

	// Padded
	if fh.Flag(3) {
		padLength = int(data[0])
		fmt.Printf("\x1b[32m(Flag)\x1b[0m Padding %d\n", padLength)
		totRead += 1
	}
	// Priority
	if fh.Flag(5) {
		depSid := binary.BigEndian.Uint32(data[totRead:])
		weight := data[totRead+4]
		fmt.Printf("\x1b[32m(Flag)\x1b[0m STREAM DEPENDENCY: %d --> %d (weight %d)\n", fh.Sid, depSid, weight)
		totRead += 5
	}
	tr, err := sess.ReadHeaders(func (k, v string) { 
		fmt.Printf("%s = %s\n", k, v)
		st.InHeaders.Add(k, v)
	}, data, totRead, padLength)
	if err != nil {
		fmt.Printf("\x1b[31m%s\x1b[0m",err)
		return err
	}
	totRead += tr
	// End Stream
	if fh.Flag(0) {
		fmt.Printf("\x1b[32m(Flag)\x1b[0m End Stream\n")
		st.State = st.State.ReceivedEndStream()
		st.Body.Close()
	}
	// End of headers
	if fh.Flag(2) {
		fmt.Printf("\x1b[32m(Flag)\x1b[0m End Headers\n")
		st.InHeaders.Closed = true
		go st.Serve(sess.Ctx)
	}
	return nil
}

func (sess *Dispatcher) ReadHeaders(cb func(k, v string), data []byte, totRead int, padLength int) (int, error) {
	for totRead < len(data) - padLength {
		hdr, numRead, err := hpack.NextHeader(data[totRead:])
		if err != nil {
			return 0, err
		}
		fmt.Printf("\x1b[32m[INFO]\x1b[0mProcess header: %s\n", hdr)
		totRead += numRead
		k, v, err := hdr.Resolve(sess.Ctx.incomingHeaderTable)
		if err != nil {
			return 0, err
		}
		cb(k, v)
		if hdr.ShouldIndex() {
			fmt.Printf("\x1b[31m[INFO]\x1b[0m: INDEX HEADER: %s\n", hdr)
			sess.Ctx.incomingHeaderTable.Insert(k, v)
		}
	}
	fmt.Println(sess.Ctx.incomingHeaderTable)
	return totRead, nil
}

func (sess *Dispatcher) DoRequest(sid frame.Sid) error {
	stream := sess.Stream(sid)
	stream.InHeaders.Closed = true
	return nil
}
