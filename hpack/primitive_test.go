package hpack

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeInteger(t *testing.T) {
	cases := []struct{D string; P int; E uint32; Enr int;}{
		{"\x47", 6, 7, 1},
		{"\x47", 7, 71, 1},
		{"\x07\x0e", 3, 21, 2},
		{"\x07\x83\x01", 3, 138, 3},
		{"\x07\x83\x01\xff", 3, 138, 3},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%#v,%d-->%d", c.D, c.P, c.E), func(t *testing.T) {
			assert := assert.New(t)
			n, numRead, err := DecodeInteger([]byte(c.D), c.P)
			assert.NoError(err)
			assert.Equal(c.E, n)
			assert.Equal(c.Enr, numRead)
		})
	}
}

func TestDecodeInteger_Error(t *testing.T) {
	_, _, err := DecodeInteger([]byte("\x07\x83"), 3)
	assert.Error(t, err)
}

func TestEncodeInteger(t *testing.T) {
	cases := []struct{N uint32; P int; E string}{
		{127, 7, "\x7f"},
		{127, 5, "\x1f\x60"},
		{0xffffffff, 7, "\x7f\x80\xff\xff\xff\x0f"},
		{255, 8, "\xff"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d,%d", c.N, c.P), func(t *testing.T) {
			data := EncodeInteger(c.N, c.P )
			// mask the non-prefix bits as we don't care about them
			data[0] &= (1 << c.P) - 1
			assert.Equal(t, c.E, string(data)) 
		})
	}
}

func TestEncodeString(t *testing.T) {
	data := "*/*"
	encoded := EncodeString([]byte(data))
	assert.Equal(t, "\x03*/*", string(encoded))
}

func TestEncodeStringHuffman(t *testing.T) {
	data := "racecar"
	encoded := EncodeString([]byte(data))

	exp := "\x85\xb0\x64\x29\x07\x67"
	assert.Equal(t, exp, string(encoded))
}

func TestDecodeStringNoHuffman(t *testing.T) {
	cases := []struct{Name string; D string; ENumRead int; Exp string}{
		{"NoHuffman", "\x02BRUH", 3, "BR"},
		{"Huffman", "\x82\x8c\xbf", 3, "be"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%v, %d", c.D, c.ENumRead), func(t *testing.T) {
			decoded, nRead, err := DecodeString([]byte(c.D))
			assert.NoError(t, err)
			assert.Equal(t, c.ENumRead, nRead)
			assert.Equal(t, c.Exp, string(decoded))
		})
	}
}

