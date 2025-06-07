package frame

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFramer(t *testing.T) {
	stream := 
		"\x00\x00\x0c\x00\x01\x00\x00\x00\x02Hello, world\x00\x00\x02\x01\x10\x00\x00\x00\x0a\xab\xcd"

	framer := NewFramer(strings.NewReader(stream))

	var (
		fr *Frame
		err error
	)

	fr, err = framer.ReadFrame()
	assert.NoError(t, err)
	assert.EqualValues(t, Sid(2), fr.FrameHeader.Sid)
	assert.EqualValues(t, 12, fr.FrameHeader.Length)
	assert.EqualValues(t, FrameData, fr.FrameHeader.Type)
	assert.True(t, fr.FrameHeader.Flag(0))

	assert.EqualValues(t, "Hello, world", string(fr.Data))
	
	fr, err = framer.ReadFrame()
	assert.NoError(t, err)
	assert.EqualValues(t, Sid(10), fr.FrameHeader.Sid)
	assert.EqualValues(t, 2, fr.FrameHeader.Length)
	assert.EqualValues(t, FrameHeaders, fr.FrameHeader.Type)
	assert.True(t, fr.FrameHeader.Flag(4))

	assert.EqualValues(t, "\xab\xcd", string(fr.Data))
}
