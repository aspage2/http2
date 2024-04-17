package hpack

import (
	"errors"
	"bufio"
)

var LookupError = errors.New("attempt to index nonexistant field")

type HeaderParser struct {
	DT *DynamicTable
	pp *PrimitiveParser
}

func NewHeaderParser(tree *HuffmanTree) *HeaderParser {
	return &HeaderParser{
		DT: NewDynamicTable(),
		pp: NewPrimitiveParser(tree),
	}
}

func (hp *HeaderParser) IndexHeader(rd *bufio.Reader) (string, string, error) {
	n, _, err := hp.pp.DecodeInteger(rd, 7)
	if err != nil {
		return "", "", err
	}
	k, v, ok := hp.DT.Lookup(int(n))
	if !ok {
		return "", "", LookupError
	}
	return k, v, nil
}

func (hp *HeaderParser) LiteralHeaderWithIncrementalIndexing(rd *bufio.Reader) (string, string, error) {
	n, _, err := hp.pp.DecodeInteger(rd, 6)
	if err != nil {
		return "", "", err
	}

	// If n is 0, there are two strings in the header payload.
	// The first is the header key, followed by the header value.
	var key string
	if n == 0 {
		data, _, _, err := hp.pp.DecodeString(rd)
		if err != nil {
			return "", "", err
		}
		key = string(data)
	} else {
		k, _, ok := hp.DT.Lookup(int(n))
		if !ok {
			return "", "", LookupError
		}
		key = k
	}
	value, _, _, err := hp.pp.DecodeString(rd)
	if err != nil {
		return "", "", err
	}
	hp.DT.Insert(key, string(value))
	return key, string(value), nil
}

func (hp *HeaderParser) NextHeader(rd *bufio.Reader) (string, string, error) {
	firstOctet, err := rd.ReadByte()
	if err != nil {
		return "", "", err 
	}
	_ = rd.UnreadByte()

	// Indexed Header Field representation
	if firstOctet & 0b10000000 != 0 {
		return hp.IndexHeader(rd)
	}
	if firstOctet & 0b01000000 != 0 {
		return hp.LiteralHeaderWithIncrementalIndexing(rd)
	}
	return "", "", errors.New("unrecognized header")
}
