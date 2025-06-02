package session

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"http2/frame"
	"http2/hpack"
	"http2/session/settings"
	"io"
	"sync"
)

// A ConnectionContext contains global state information about a single
// connection.
type ConnectionContext struct {
	context.Context
	incoming            io.Reader
	incomingHeaderTable *hpack.HeaderLookupTable

	outlock             *sync.Mutex
	outgoing            io.Writer
	outgoingHeadertable *hpack.HeaderLookupTable

	cancel context.CancelFunc

	Settings settings.SettingsList

	Handler Handler
}

func NewConnectionContext(in io.Reader, out io.Writer, handler Handler) *ConnectionContext {
	ctx, cancel := context.WithCancel(context.Background())

	ret := &ConnectionContext{
		incoming:            in,
		incomingHeaderTable: hpack.NewHeaderLookupTable(),

		outgoing:            out,
		outlock:             new(sync.Mutex),
		outgoingHeadertable: hpack.NewHeaderLookupTable(),

		Context: ctx,
		cancel:  cancel,

		Handler: handler,
	}
	return ret
}

func (this *ConnectionContext) SendFrame(fh *frame.FrameHeader, data []byte) error {
	this.outlock.Lock()
	defer this.outlock.Unlock()

	fmt.Printf("\x1b[31mSend Frame\x1b[0m %s\n", fh)
	if data != nil && len(data) > 1 {
		fmt.Printf(hex.Dump(data[:min(len(data), 1024)]))
	}
	if err := fh.Marshal(this.outgoing); err != nil {
		return err
	}
	if data == nil || len(data) == 0 {
		return nil
	}
	_, err := io.Copy(this.outgoing, bytes.NewReader(data))
	return err
}
