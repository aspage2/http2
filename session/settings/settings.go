package settings

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

//go:generate stringer -type SettingsType
type Type uint16

const (
	// The maximum size of the header lookup table
	HeaderTableSize Type = iota + 1

	// Whether the client or server support server-push
	EnablePush

	// The maximum number of streams allowed to be in an "open"
	// or "reserved" state
	MaxConcurrentStreams

	InitialWindowSize

	// the maximum HTTP frame size supported during the session
	MaxFrameSize

	// The maximum number of headers accepted by the server.
	MaxHeaderListSize
)

// STGS_ACK is a flag used in a settings
const STGS_ACK = 0x01

type setting struct {
	Type  Type
	Value uint32
}

func Default(typ Type) (val uint32, ok bool) {
	ok = true
	switch typ {
	case HeaderTableSize:
		val = 4096
	case EnablePush:
		val = 1
	case InitialWindowSize:
		val = 16535
	case MaxFrameSize:
		val = 16384
	default:
		ok = false
	}
	return
}

// A list of HTTP settings defined by RFC 7540.
type SettingsList struct {
	Settings []setting
}

// Inserts a new setting into the Settings list.
func (sl *SettingsList) Put(typ Type, value uint32) {
	for i := range sl.Settings {
		st := &sl.Settings[i]
		if st.Type == typ {
			st.Value = value
			return
		}
	}
	sl.Settings = append(sl.Settings, setting{typ, value})
}

func (sl *SettingsList) Get(typ Type) (uint32, bool) {
	for _, s := range sl.Settings {
		if s.Type == typ {
			return s.Value, true
		}
	}
	return 0, false
}

// Parse a settings list from a Settings frame payload
func SettingsListFromFramePayload(data []uint8) *SettingsList {
	if len(data)%6 != 0 {
		panic("SettingsListFromPaylaod needs a multiple of 6 payload")
	}
	ret := &SettingsList{}
	for i := 0; i < len(data); i += 6 {
		typ := binary.BigEndian.Uint16(data[i : i+2])
		value := binary.BigEndian.Uint32(data[i+2 : i+6])
		ret.Put(Type(typ), value)
	}
	return ret
}

func (sl *SettingsList) ToPayload() []byte {
	ret := make([]byte, len(sl.Settings)*6)
	i := 0
	for _, setting := range sl.Settings {
		binary.BigEndian.PutUint16(ret[i:], uint16(setting.Type))
		binary.BigEndian.PutUint32(ret[i+2:], setting.Value)
	}
	return ret
}

func (sl *SettingsList) String() string {
	var sb strings.Builder
	typs := []Type{
		HeaderTableSize,
		EnablePush,
		MaxConcurrentStreams,
		InitialWindowSize,
		MaxFrameSize,
		MaxHeaderListSize,
	}	

	for _, t := range typs {
		v, ok := sl.Get(t)
		var (
			c rune
			m string
		)
		if !ok {
			c = ' '
			v, ok = Default(t)
		} else {
			c = 'X'
		}

		if !ok {
			m = "N/A"
		} else {
			m = strconv.Itoa(int(v))
		}
		fmt.Fprintf(&sb, "(%c) %s = %s\n", c, t, m)
	}
	return sb.String()
}
