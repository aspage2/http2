package hpack

import (
	"bufio"
	"io"
)

func oneMask(n int) uint8 {
	switch n {
	case 0:
		return 0b00000000
	case 1:
		return 0b00000001
	case 2:
		return 0b00000011
	case 3:
		return 0b00000111
	case 4:
		return 0b00001111
	case 5:
		return 0b00011111
	case 6:
		return 0b00111111
	case 7:
		return 0b01111111
	case 8:
		return 0b11111111
	default:
		panic("cannot make a one-mask of more than 8 bits.")
	}
}

type PrimitiveParser struct {
	ht *HuffmanTree
}

func NewPrimitiveParser(ht *HuffmanTree) *PrimitiveParser {
	return &PrimitiveParser{ht}
}

// EncodeInteger writes the integer i as an hpack-encoded octet sequence to the given writer.
// Client code can set the values of the prefix bits in the first octet using the prefixFlags
// argument. The value of prefix flags is l-shifted by the size of the hpack bit prefix, then
// bit-set into place in the string.
//
// Returns the number of bytes written to the buffer, and escalates all errors from the buffer.
//
// Example: EncodeInteger(26, 4, 0b0010, writer)
//
// |--prefixFlags--|--4 bit mask---|
// +---+---+---+---+---+---+---+---+
// | 0 | 0 | 1 | 0 | 1 | 1 | 1 | 1 |
// | 0 | 0 | 0 | 0 | 1 | 0 | 1 | 1 |
// +---+---+---+---+---+---+---+---+
// |----------26-15 (11)-----------|
func (hp *PrimitiveParser) EncodeInteger(i uint32, n int, prefixFlags uint8, w *bufio.Writer) (int, error) {
	// A one-mask of size n is the same as 2^n - 1 in decimal.
	prefix := oneMask(n)
	prefixFlags = prefixFlags << n

	var c uint8
	if i < uint32(prefix) { 
		c = uint8(i) | prefixFlags
	} else {
		c = prefix | prefixFlags
	}
	if err := w.WriteByte(c); err != nil {
		return 0, err
	}
	if i < uint32(prefix) {
		return 1, nil
	}
	numWritten := 1
	i -= uint32(prefix)
	for i >= 0x80 {
		octet := uint8(i & 0xff) | 0x80
		if err := w.WriteByte(octet); err != nil {
			return numWritten, err
		}
		numWritten += 1
		i >>= 7
	}
	octet := uint8(i & 0xff) & 0xef
	if err := w.WriteByte(octet); err != nil {
		return numWritten, err
	}
	return numWritten + 1, nil
}

func (hp *PrimitiveParser) DecodeInteger(octets *bufio.Reader, n int) (uint32, uint32, error) {
	prefix := oneMask(n)

	b, err := octets.ReadByte()
	if err != nil {
		return 0, 0, err
	}

	prefixOctet := b & prefix
	if prefixOctet < prefix {
		return uint32(prefixOctet), 1, nil
	}

	shift := 0
	var (
		rest uint32
		numOctets uint32
	)
	for {
		b, err := octets.ReadByte()
		if err != nil {
			return 0, 0, err
		}
		numOctets += 1
		rest |= uint32(b & 0x7f) << shift
		shift += 7
		if b & 0x80 == 0 {
			break
		}
	}
	// Add back the 2^N - 1 prefix to the result.
	return uint32(prefix) + rest, numOctets, nil
}

func (hp *PrimitiveParser) DecodeString(octets *bufio.Reader) ([]uint8, bool, uint32, error) {
	first, err := octets.Peek(1)
	if err != nil {
		return nil, false, 0, err
	}

	isHuffman := first[0] & 0x80 != 0
	l, off, err := hp.DecodeInteger(octets, 7)
	if err != nil {
		return nil, false, 0, err
	}

	ret := make([]uint8, l)
	_, err = io.ReadFull(octets, ret)
	if err != nil {
		return nil, false, 0, err
	}
	if isHuffman {
		ret = hp.ht.Decode(ret)
	}
	return ret, isHuffman, off + l, nil
}

func (hp *PrimitiveParser) EncodeString(data []uint8, huffman bool, w *bufio.Writer) (int, error) {
	var ret []uint8
	val := data
	var flag uint8 = 0
	if huffman {
		val = hp.ht.Encode(data)
		flag = 1
	}
	bytesWritten, err := hp.EncodeInteger(uint32(len(val)), 7, flag, w)
	if err != nil {
		return 0, err
	}
	if huffman {
		ret[0] |= 0x80
	}
	return bytesWritten, nil
}
