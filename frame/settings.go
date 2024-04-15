package frame

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

type SettingType uint16

const (
	// TODO: Implicitly assigning values with iota is kind of icky.
	SettingsUndefined SettingType = iota

	// The HPACK header compression method includes a "reference"
	// table for storing frequently-used header pairs. The table
	// is split into a predefined "static" portion (formally defined
	// in the RFC in Appendix A) and a "dynamic" portion that is
	// incrementally updated during the span of a connection. This
	// table enables actors to specify header pairs with a single
	// index value rather than use bandwidth serializing text data
	// onto the wire.
	//
	// This setting provides the ability for an actor to declare
	// the size of the dynamic table.
	SettingsHeaderTableSize

	SettingsEnablePush

	// The max number of streams that can be serviced during the
	// life of the connection.
	SettingsMaxConcurrentStreams
	SettingsInitialWindowSize
	SettingsMaxFrameSize
	SettingsMaxHeaderListSize
)

func (st SettingType) String() string {
	switch st {
	case SettingsHeaderTableSize:
		return "SETTINGS_HEADER_TABLE_SIZE"
	case SettingsEnablePush:
		return "SETTINGS_ENABLE_PUSH"
	case SettingsMaxConcurrentStreams:
		return "SETTINGS_MAX_CONCURRENT_STREAMS"
	case SettingsInitialWindowSize:
		return "SETTINGS_INITIAL_WINDOW_SIZE"
	case SettingsMaxFrameSize:
		return "SETTINGS_MAX_FRAME_SIZE"
	case SettingsMaxHeaderListSize:
		return "SETTINGS_MAX_HEADER_LIST_SIZE"
	default:
		return "UNDEFINED"
	}
}

type SettingsPayload struct {
	SettingTypes  []SettingType
	SettingValues []uint32
	IsAck         bool
}

func NewSettingsPayload(isAck bool) *SettingsPayload {
	return &SettingsPayload{IsAck: isAck}
}

func SettingsFromFrame(fr *Frame) (*SettingsPayload, error) {
	var ret SettingsPayload

	if fr.Type != 0x4 {
		return nil, errors.New("not a settings payload")
	}

	ret.IsAck = fr.Flags == 0x1
	numSettings := fr.Length / 6
	ret.SettingTypes = make([]SettingType, numSettings)
	ret.SettingValues = make([]uint32, numSettings)

	j := 0
	for i := uint32(0); i < numSettings; i++ {
		ret.SettingTypes[i] = SettingType(binary.BigEndian.Uint16(fr.Data[j:]))
		ret.SettingValues[i] = binary.BigEndian.Uint32(fr.Data[j+2:])
		j += 6
	}
	return &ret, nil
}

func (sf *SettingsPayload) ToFrame(fr *Frame) {
	fr.Type = FrameTypeSettings
	if sf.IsAck {
		fr.Flags = 0x1
	} else {
		fr.Flags = 0x0
	}
	// TODO: Should checks be in place for
	// ACK settings frames with settings included?
	fr.Length = uint32(len(sf.SettingTypes) * 6)

	fr.Data = make([]uint8, fr.Length)
	j := 0
	for i := 0; i < len(sf.SettingTypes); i++ {
		binary.BigEndian.PutUint16(fr.Data[j:], uint16(sf.SettingTypes[i]))
		binary.BigEndian.PutUint32(fr.Data[j+2:], sf.SettingValues[j])
		j += 6
	}
}

func (sf *SettingsPayload) String() string {
	var sb strings.Builder

	sb.WriteString("Type: SETTINGS")
	if sf.IsAck {
		sb.WriteString("(ACK) ")
	}
	sb.WriteRune('\n')

	for i := 0; i < len(sf.SettingTypes); i++ {
		fmt.Fprintf(&sb, "%s = %d", sf.SettingTypes[i], sf.SettingValues[i])
		if i != len(sf.SettingTypes) - 1 {
			sb.WriteRune('\n')
		}
	}
	return sb.String()
}

func (sf *SettingsPayload) Put(typ SettingType, value uint32) {
	sf.SettingTypes = append(sf.SettingTypes, typ)
	sf.SettingValues = append(sf.SettingValues, value)
}
