package hpack

import (
	"bytes"
	"io"
	"strings"
)

type HeaderList struct {
	data bytes.Buffer
	tbl  *HeaderLookupTable
}

func NewHeaderList(table *HeaderLookupTable) *HeaderList {
	return &HeaderList{
		tbl: table,
	}
}

func (hl *HeaderList) Put(k, v string) {
	k = strings.ToLower(k)
	v = strings.ToLower(v)

	ind, justKey := hl.tbl.Find(k, v)
	if ind > 0 && !justKey {
		hl.data.Write(IndexedHeader(ind).Encode())
		return
	}

	hdr := new(LiteralHeader)
	switch k {
	case "authorization":
		hdr.Type = NeverIndex
	case ":path":
		hdr.Type = NoIndex
	default:
		hdr.Type = IncrementalIndex
	}
	if ind > 0 {
		hdr.KeyIndex = uint32(ind)
	} else {
		hdr.KeyLiteral = k
	}
	hdr.ValueLiteral = v
	hl.data.Write(hdr.Encode())
}

func (hl *HeaderList) Dump() []byte {
	data, _ := io.ReadAll(&hl.data)
	return data
}
