package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandle_Error(t *testing.T) {
	sameLengthWrongChars := make([]byte, len(ClientPreface))
	copy(sameLengthWrongChars, ClientPreface)
	sameLengthWrongChars[0] = 'I'
	cases := [][]byte{
		ClientPreface[:10],
		sameLengthWrongChars,
	}

	for _, c := range cases {
		t.Run(string(c), func(t *testing.T) {
			err := ConsumePreface(bytes.NewReader(c))
			assert.Equal(t, UnexpectedPreface, err)
		})
	}
}
