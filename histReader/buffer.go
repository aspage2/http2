package histReader

import (
	"bytes"
	"encoding/hex"
	"io"
)

type HistReader struct {
	rd   io.Reader
	hist bytes.Buffer
}

func NewHistReader(rd io.Reader) *HistReader {
	return &HistReader{rd: rd}
}

func (hr *HistReader) Read(b []byte) (int, error) {
	n, err := hr.rd.Read(b)
	if err != nil {
		return 0, err
	}
	hr.hist.Write(b)
	return n, nil
}

func (hr *HistReader) Dump() string {
	data, _ := io.ReadAll(&hr.hist)

	lastNonZeroByte := 0
	for i, b := range data {
		if b != 0 {
			lastNonZeroByte = i
		}
	}
	numRows := lastNonZeroByte/16 + 1
	return hex.Dump(data[:numRows*16])
}

func (hr *HistReader) Clear() {
	hr.hist.Reset()
}
