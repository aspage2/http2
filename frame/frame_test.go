package frame

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)


func TestFrameHeaderUnmarshal(t *testing.T) {
	data := "\x00\x00\x0a\x01\xaa\x00\x00\xab\xcd"

	var fh FrameHeader
	fh.Unmarshal(strings.NewReader(data))

	assert.EqualValues(t, 10, fh.Length)
	assert.EqualValues(t, FrameHeaders, fh.Type)
	assert.EqualValues(t, 0b10101010, fh.Flags)
	assert.EqualValues(t, Sid(0xabcd), fh.Sid)
}

func TestFrameHeaderMarshal(t *testing.T) {
	var fh FrameHeader

	fh.Sid = 33
	fh.Flags = 0b00001000
	fh.Type = FrameWindowUpdate
	fh.Length = 20

	buf := bytes.NewBuffer(nil)
	assert.NoError(t, fh.Marshal(buf))

	assert.EqualValues(t, "\x00\x00\x14\x08\x08\x00\x00\x00\x21", buf.Bytes())
}

func TestFrameHeaderFlag(t *testing.T) {
	var fh FrameHeader

	fh.Flags = 0b00100001

	exp := []bool{true, false, false, false, false, true, false, false}

	for i := range exp {
		assert.Equal(t, exp[i], fh.Flag(i))
	}
}

func TestFrameHeaderFlagPanic(t *testing.T) {
	var fh FrameHeader

	assert.Panics(t, func() {
		fh.Flag(100)
	})
}
