package hpack

import (
	"errors"
)

func oneMask(n int) uint8 {
	return (1 << n) - 1
}

// DecodeInteger decodes an HPACK-encoded integer. HPACK-encoded integers
// get encoded into two "parts":
//
// * The first `prefixLength` bits of the first octet
// * A sequence of octets that make up the remainder of the number.
//
// To Decode an Hpack-encoded integer:
//
//  1. take the `prefixLength` least-significant bits from the first octet
//  2. take the next sequence of octets up to (and including) the octet with its
//     bit #7 set. Concatenate the 7 least-significant bits of the sequence in
//     little-endian order.
//  3. Add the value from #1 and the value from #2.
//
// HPACK-encoded integers are done this way to allow for an integer to
// start midway through an octet, leaving room for any flags or prefixes
// that
func DecodeInteger(data []uint8, prefixLength int) (uint32, int, error) {
	prefixMask := oneMask(prefixLength)
	prefix := data[0] & prefixMask

	if prefix < prefixMask {
		return uint32(prefix), 1, nil
	}

	var ret uint32
	var shift int
	i := 1
	for i < len(data) {
		ret |= uint32(data[i]&0x7f) << shift
		if data[i]&0x80 == 0 {
			break
		}
		shift += 7
		i++
	}
	if i == len(data) {
		return 0, 0, errors.New("invalid hpack integer")
	}
	return ret + uint32(prefix), i + 1, nil
}

func EncodeInteger(n uint32, prefixLength int) []byte {
	prefixMask := oneMask(prefixLength)
	if n <= uint32(prefixMask) {
		return []byte{uint8(n)}
	}

	ret := append(make([]byte, 0, 5), prefixMask)
	rest := n - uint32(prefixMask)
	for rest > 0 {
		ret = append(ret, (uint8(rest)&0x7f)|0x80)
		rest >>= 7
	}
	// Set the last bit to signify the end of the integer
	ret[len(ret)-1] &= 0x7F
	return ret
}

// DecodeString decodes an hpack-encoded string.
// Strings begin with an hpack-encoded integer of prefix 7
// which represent the length of the string data on the wire.
// The string data follows right after the HPACK integer.
// If the MSB of the first octet is 1, the string is huffman
// coded with the canonical huffman code given in RFC 7541.
func DecodeString(data []uint8) ([]byte, int, error) {
	isHuffmanEncoded := data[0]&0x80 != 0

	dataLength, numRead, err := DecodeInteger(data, 7)
	if err != nil {
		return nil, 0, err
	}
	stringData := data[numRead : numRead+int(dataLength)]
	if isHuffmanEncoded {
		stringData = HpackHuffmanTree.Decode(stringData)
	}
	return stringData, int(dataLength) + numRead, nil
}

func EncodeString(data []byte) []byte {
	huffEncoded := HpackHuffmanTree.Encode(data)
	shouldUseHuffman := len(huffEncoded) < len(data)

	payloadToEncode := data
	if shouldUseHuffman {
		payloadToEncode = huffEncoded
	}

	lenEncoded := EncodeInteger(uint32(len(payloadToEncode)), 7)
	if shouldUseHuffman {
		lenEncoded[0] |= 0x80
	}
	ret := make([]uint8, len(lenEncoded)+len(payloadToEncode))
	n := copy(ret, lenEncoded)
	copy(ret[n:], payloadToEncode)

	return ret
}
