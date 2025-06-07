package session

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"http2/frame"
	"http2/hpack"
	"http2/session/settings"
)

// A Dispatcher object represents an open connection
// between this server and a client.
type Dispatcher struct {
	Framer     *frame.Framer
	Ctx        *ConnectionContext
	lastStream frame.Sid
	Streams    map[frame.Sid]*Stream
}

func NewDispatcher(ctx *ConnectionContext, framer *frame.Framer) *Dispatcher {
	var sess Dispatcher
	sess.Ctx = ctx
	sess.Framer = framer
	sess.lastStream = 0
	sess.Streams = make(map[frame.Sid]*Stream)
	return &sess
}

func (sess *Dispatcher) initialHandshake() error {
	err := sess.Framer.ConsumePreface()
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
	fr, ok, err := sess.ExpectFrame(frame.FrameSettings, 0)
	if err != nil {
		return err
	} else if !ok {
		return errors.New("Unexpected frame")
	}
	sl := settings.SettingsListFromFramePayload(fr.Data)
	fmt.Println("---(INITIAL CLIENT SETTINGS)---")
	fmt.Print(sl)
	fmt.Println("-------------------------------")
	globalStream.SendFrame(frame.FrameSettings, settings.STGS_ACK, nil)
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
		fr  *frame.Frame
		err error
	)
	for {
		fr, err = sess.Framer.ReadFrame()
		if err != nil {
			break
		}
		err = sess.Dispatch(fr)
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

func (sess *Dispatcher) ExpectFrame(typ frame.FrameType, sid frame.Sid) (*frame.Frame, bool, error) {
	fr, err := sess.Framer.ReadFrame()
	if err != nil {
		return nil, false, err
	}
	return fr, typ == fr.FrameHeader.Type && sid == fr.FrameHeader.Sid, nil
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
func (sess *Dispatcher) Dispatch(fr *frame.Frame) error {
	fh := fr.FrameHeader
	data := fr.Data
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
	case frame.FrameWindowUpdate:
		d := binary.BigEndian.Uint32(data[:4])
		d &= ^uint32(1 << 31)
		fmt.Printf("Client can receive an extra \x1b[33m%d\x1b[0m octets\n", d)
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
		LastSid:   sess.lastStream,
		Reason:    reason,
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
	tr, err := sess.ReadHeaders(func(k, v string) {
		fmt.Printf("%s = %s\n", k, v)
		st.InHeaders.Add(k, v)
	}, data, totRead, padLength)
	if err != nil {
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
	for totRead < len(data)-padLength {
		hdr, numRead, err := hpack.NextHeader(data[totRead:])
		if err != nil {
			return 0, err
		}
		totRead += numRead
		k, v, err := hdr.Resolve(sess.Ctx.incomingHeaderTable)
		if err != nil {
			return 0, err
		}
		cb(k, v)
		if hdr.ShouldIndex() {
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
