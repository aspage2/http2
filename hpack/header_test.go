package hpack

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexedHeaderDontIndex(t *testing.T) {
	assert.False(t, IndexedHeader(33).ShouldIndex())
}

//	8   0   f   d
//
// 0000000011111101
func TestIndexedHeaderEncode(t *testing.T) {

	cases := []struct {
		C uint32
		E string
	}{
		{C: 32, E: "\xa0"},
		{C: 16000 + 127, E: "\xff\x80\x7d"},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%#v", c.E), func(t *testing.T) {
			data := IndexedHeader(c.C).Encode()
			assert.Equal(t, len(c.E), len(data))

			for i := 0; i < len(data); i++ {
				assert.Equal(t, c.E[i], data[i])
			}
			assert.NotEqual(t, data[0]&0x80, 0)
		})
	}
}
