package hpack

import "encoding/binary"

func oneMask(n int) uint32 {
	var maxInPrefix uint32
	for j := 0; j < n; j ++ {
		maxInPrefix = (maxInPrefix << 1) | 1
	}
	return maxInPrefix
}

// EncodeInteger creates an octet representation of an integer.
func EncodeInteger(i uint32, n int) []byte {
	// 2^n - 1
	maxInPrefix := oneMask(n)

	if i < maxInPrefix {
		octet := binary.BigEndian.AppendUint32(nil, i)
		return octet[3:]
	}
	var octets []uint8
	octets = append(octets, uint8(maxInPrefix))
	i -= maxInPrefix
	for i >= 0x80 {
		octets = append(octets, uint8(i & 0xFF) | 0x80)
		i >>= 7
	}
	octets = append(octets, uint8(i & 0xFF) & 0xEF)
	return octets
}

func DecodeToInteger(octets []uint8, n int) (uint32, uint32) {
	prefix := oneMask(n)

	prefixOctet := uint32(octets[0]) & prefix
	if prefixOctet < prefix {
		return prefixOctet, 1
	}

	shift := 0
	var (
		rest uint32
		numOctets uint32
	)
	for j := 1; j < len(octets); j ++ {
		o := octets[j]
		numOctets += 1
		rest |= uint32(o & 0x7f) << shift
		shift += 7
		if o & 0x80 == 0 {
			break
		}
	}
	// Add back the 2^N - 1 prefix to the result.
	return prefix + rest, numOctets
}

func DecodeString(octets []uint8) ([]uint8, bool, uint32) {
	isHuffman := octets[0] & 0x80 != 0
	l, off := DecodeToInteger(octets, 7)
	ret := octets[off:off+l]
	if isHuffman {
		ret = NewHuffmanTree().Decode(ret)
	}
	return ret, isHuffman, off + l
}

