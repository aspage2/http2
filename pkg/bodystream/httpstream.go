package bodystream

import (
	"bytes"
	"io"
	"sync"
)

// A BodyStream is a single writer, single
// reader byte stream.
type BodyStream struct {
	buf *bytes.Buffer

	isClosed bool

	mu *sync.Mutex
	cv *sync.Cond
}

func NewBodyStream() *BodyStream {
	var bs BodyStream
	bs.buf = new(bytes.Buffer)
	bs.mu = new(sync.Mutex)
	bs.cv = sync.NewCond(bs.mu)
	return &bs
}

func (st *BodyStream) Close() error {
	st.mu.Lock()
	st.isClosed = true
	st.mu.Unlock()
	st.cv.Signal()
	return nil
}

func (st *BodyStream) Read(data []byte) (int, error) {
	st.mu.Lock()
	for !st.isClosed && st.buf.Len() <= 0 {
		st.cv.Wait()
	}
	defer st.mu.Unlock()
	if st.buf.Len() > 0 {
		return st.buf.Read(data)
	}
	return 0, io.EOF
}

func (st *BodyStream) Write(data []byte) (int, error) {
	st.mu.Lock()
	ret, err := st.buf.Write(data)
	st.mu.Unlock()
	st.cv.Signal()
	return ret, err
}

