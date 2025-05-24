package session

import (
	"encoding/binary"
)

//go:generate stringer -type SettingsType
type SettingsType uint16

const (
	// The maximum size of the header lookup table
	SettingsHeaderTableSize SettingsType = iota + 1

	// Whether the client or server support server-push
	SettingsEnablePush

	// The maximum number of streams allowed to be in an "open"
	// or "reserved" state
	SettingsMaxConcurrentStreams

	SettingsInitialWindowSize

	// the maximum HTTP frame size supported during the session
	SettingsMaxFrameSize

	// The maximum number of headers accepted by the server.
	SettingsMaxHeaderListSize
)

// STGS_ACK is a flag used in a settings
const STGS_ACK = 0x01

type setting struct {
	Type  SettingsType
	Value uint32
}

// A list of HTTP settings defined by RFC 7540.
type SettingsList struct {
	Settings []setting
}

// Inserts a new setting into the Settings list.
func (sl *SettingsList) Put(typ SettingsType, value uint32) {
	sl.Settings = append(sl.Settings, setting{typ, value})
}

// Parse a settings list from a Settings frame payload
func SettingsListFromFramePayload(data []uint8) *SettingsList {
	if len(data)%6 != 0 {
		panic("SettingsListFromPaylaod needs a multiple of 6 payload")
	}
	numSettings := len(data) / 6
	var ret SettingsList
	ret.Settings = make([]setting, numSettings)
	j := 0
	for i := range numSettings {
		ret.Settings[i].Type = SettingsType(
			binary.BigEndian.Uint16(data[j : j+2]),
		)
		ret.Settings[i].Value = binary.BigEndian.Uint32(data[j+2 : j+6])
		j += 6
	}
	return &ret
}
