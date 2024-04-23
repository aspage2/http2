package session

import (
	"encoding/binary"
)

//go:generate stringer -type SettingsType
type SettingsType uint16

const (
	SettingsHeaderTableSize SettingsType = iota + 1
	SettingsEnablePush
	SettingsMaxConcurrentStreams
	SettingsInitialWindowSize
	SettingsMaxFrameSize
	SettingsMaxHeaderListSize
)

type setting struct {
	Type  SettingsType
	Value uint32
}

type SettingsList struct {
	Settings []setting
}

func (sl *SettingsList) Put(typ SettingsType, value uint32) {
	sl.Settings = append(sl.Settings, setting{typ, value})
}

func SettingsListFromFramePayload(data []uint8) *SettingsList {
	if len(data)%6 != 0 {
		panic("SettingsListFromPaylaod needs a multiple of 6 payload")
	}
	numSettings := len(data) / 6
	var ret SettingsList
	ret.Settings = make([]setting, numSettings)
	j := 0
	for i := 0; i < numSettings; i++ {
		ret.Settings[i].Type = SettingsType(
			binary.BigEndian.Uint16(data[j : j+2]),
		)
		ret.Settings[i].Value = binary.BigEndian.Uint32(data[j+2 : j+6])
		j += 6
	}
	return &ret
}
