package hpack

func oneMask(n int) uint8 {
	switch n {
	case 0:
		return 0
	case 1:
		return 0b1
	case 2:
		return 0b11
	case 3:
		return 0b111
	case 4:
		return 0b1111
	case 5:
		return 0b11111
	case 6:
		return 0b111111
	case 7:
		return 0b1111111
	case 8:
		return 0b11111111
	}
	return 0
}

// DecodeInteger decodes an HPACK-encoded integer. HPACK-encoded integers
// get encoded into two "parts":
//
// * The first `prefixLength` bits of the first octet
// * A sequence of octets that make up the remainder of the number.
//
// To Decode an Hpack-encoded integer:
//
// 1. take the `prefixLength` least-significant bits from the first octet
// 2. take the next sequence of octets up to (and including) the octet with its
//    bit #7 set. Concatenate the 7 least-significant bits of the sequence in
//    network-byte order.
// 3. Add the value from #1 and the value from #2.
//
// HPACK-encoded integers are done this way to allow for an integer to
// start midway through an octet, leaving room for any flags or prefixes
// that 
func DecodeInteger(data []uint8, prefixLength int) (uint32, int, error) {
	prefixMask := oneMask(prefixLength)

	if prefix := data[0] & prefixMask; prefix < prefixMask {
		return uint32(prefix), 1, nil
	}

	var ret uint32
	var retBit int
	i := 0
	for {
		ret |= 
		if data[i]&0x80 == 0 {
			break
		}
		i ++
	}
}
