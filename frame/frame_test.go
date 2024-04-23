package frame

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)


func TestFrameHeader_Flags(t *testing.T) {
	fh := new(FrameHeader)

	fh.Flags = 0b00101000

	for i := 0; i < 8; i ++ {
		if i == 5 || i == 3 {
			assert.True(t, fh.Flag(i))
		} else {
			assert.False(t, fh.Flag(i))
		}
	}
}

func TestFrameHeaderUnmarshal(t *testing.T) {
	var flags uint8 = 0b00010100
	var typ FrameType = FrameHeaders

	//            |---- 1337 ------|                  |--------- 420 ---------|
	data := []byte{0x00, 0x05, 0x39, uint8(typ), flags, 0x00, 0x00, 0x01, 0xa4}

	fh := new(FrameHeader)
	assert.NoError(t, fh.Unmarshal(bytes.NewReader(data)))

	assert.Equal(t, uint32(1337), fh.Length)
	assert.Equal(t, typ, fh.Type)
	assert.Equal(t, flags, fh.Flags)
	assert.Equal(t, Sid(420), fh.Sid)
}

