package hpack

import (
	"errors"
	"fmt"
	"strings"
)

var (
	LookupIndexOutOfBounds = errors.New("lookup table out-of-bounds")
)

// A Header represents the encoded form of an HPACK header block.
type Header interface {
	Resolve(*HeaderLookupTable) (string, string, error)

	ShouldIndex() bool
}

// An IndexedHeader represents a header pair that can be found
// at the IndexHeader's value in a lookup table.
type IndexedHeader int

// IndexedHeaders are already indexed, so it doesn't make
// sense to put the values back in!
func (ih IndexedHeader) ShouldIndex() bool {
	return false
}

func (ih IndexedHeader) Resolve(table *HeaderLookupTable) (string, string, error) {
	k, v, ok := table.Lookup(int(ih))
	if !ok {
		return "", "", LookupIndexOutOfBounds
	}
	return k, v, nil
}

func (ih IndexedHeader) String() string {
	return fmt.Sprintf("Header.Indexed[%d]", ih)
}

//go:generate stringer -type=LiteralIndexType
type LiteralIndexType uint8

const (
	// Literals should be inserted into the dynamic table.
	IncrementalIndex LiteralIndexType = iota

	// Literals should not be inserted into the dynamic table,
	// for example, if a header is a long url being requested once.
	NoIndex

	// Literals should never be inserted into the dynamic table.
	// This type is normally reserved for sensitive headers like
	// session tokens, cookies and authorization tokens.
	NeverIndex
)

// A LiteralHeader represents a header pair where the value
// (and optionally the key) are string literals.
type LiteralHeader struct {
	// The Indexing type of a Literal header indicates
	// whether/not a literal header should be inserted into
	// a lookup table. If the type is either NoIndex or
	// NeverIndex, then the literal header should not be
	// indexed into a table.
	Type LiteralIndexType

	KeyIndex   uint32
	KeyLiteral string

	ValueLiteral string
}

func (lh *LiteralHeader) Resolve(table *HeaderLookupTable) (string, string, error) {
	if lh.KeyLiteral != "" {
		return string(lh.KeyLiteral), string(lh.ValueLiteral), nil
	}
	k, _, ok := table.Lookup(int(lh.KeyIndex))
	if !ok {
		return "", "", LookupIndexOutOfBounds
	}
	return k, string(lh.ValueLiteral), nil
}

func (lh *LiteralHeader) ShouldIndex() bool {
	return lh.Type == IncrementalIndex
}

func (lh *LiteralHeader) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Header.Literal(index=%s, ", lh.Type)
	if lh.KeyLiteral != "" {
		fmt.Fprintf(&sb, "k='%s', ", lh.KeyLiteral)
	} else {
		fmt.Fprintf(&sb, "k=I[%d], ", lh.KeyIndex)
	}

	fmt.Fprintf(&sb, "v='%s')", lh.ValueLiteral)
	return sb.String()
}

// NextHeader tries to extract a header from the start
// of the given octet buffer.
func NextHeader(data []uint8) (Header, int, error) {
	c := data[0]

	if c&0b10000000 != 0 {
		idx, numRead, err := DecodeInteger(data, 7)
		if err != nil {
			return nil, 0, err
		}
		return IndexedHeader(idx), numRead, nil
	}

	if c&0b01000000 != 0 {
		return literalHeader(data, IncrementalIndex, 6)
	} else if c&0b00100000 != 0 {
		fmt.Println("this is a data increase directive....")
		return nil, 1, nil
	} else if c&0b00010000 != 0 {
		return literalHeader(data, NeverIndex, 4)
	} else if c&0b11110000 == 0 {
		return literalHeader(data, NoIndex, 4)
	}
	return nil, 0, errors.New("this is bad.")
}

func literalHeader(data []uint8, typ LiteralIndexType, prefixSize int) (Header, int, error) {
	n, totalRead, err := DecodeInteger(data, prefixSize)
	if err != nil {
		return nil, 0, err
	}
	var lh LiteralHeader
	lh.Type = typ
	// when n is zero, this key is a literal.
	if n == 0 {
		s, numRead, err := DecodeString(data[totalRead:])
		if err != nil {
			return nil, 0, err
		}
		totalRead += numRead
		lh.KeyLiteral = string(s)
	} else {
		lh.KeyIndex = n
	}

	s, numRead, err := DecodeString(data[totalRead:])
	if err != nil {
		return nil, 0, err
	}
	totalRead += numRead
	lh.ValueLiteral = string(s)
	return &lh, totalRead, nil
}
