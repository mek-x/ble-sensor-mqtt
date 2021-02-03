package main

import (
	"encoding/binary"

	"github.com/go-ble/ble"
	"github.com/pkg/errors"
)

type DevData struct {
	T     float64 `json:"T"`
	H     float64 `json:"H"`
	P     float64 `json:"P"`
	BattL uint16  `json:"battLvl"`
	BattV float64 `json:"battVolt"`
	Count uint32  `json:"count"`
}

type parser func([]byte, []ble.ServiceData) (*DevData, error)

var parsers = map[string]parser{
	"ATC":   atcParser,
	"inode": inodeParser,
}

func min(x, y uint16) uint16 {
	if x <= y {
		return x
	}
	return y
}

const (
	UUID_ATC    = 0x181a
	UUID_XIAOMI = 0xFE95
)

/* ATC format:
- for UUID == 0x181a and len == 15:
uint8_t		MAC[6]; // [0] - lo, .. [6] - hi digits
int16_t		temperature; // x 0.01 degree
uint16_t	humidity; // x 0.01 %
uint16_t	battery_mv; // mV
uint8_t		battery_level; // 0..100 %
uint8_t		counter; // measurement count
uint8_t		flags;

- [not implemented] for UUID == 0x181a and len == 12:
uint8_t		MAC[6]; // [0] - hi, .. [6] - lo digits (big-endian!)
uint8_t		temperature[2]; // x 0.1 degree (big-endian!)
uint8_t		humidity; // x 1 %
uint8_t		battery_level; // 0..100 %
uint8_t		battery_mv[2]; // mV (big-endian!)
uint8_t		counter; // measurement count

- [not implemented] for UUID == 0xFE95 and len == 17:
uint16_t	ctrl;	// = 0x3050 Frame ctrl
uint16_t    dev_id; // = 0x055B	Device type
uint8_t		counter; // 0..0xff..0 measurement count
uint8_t		MAC[6];	// [0] - lo, .. [6] - hi digits
// +15: 0x0A, 0x10, 0x01, t_lv, 0x02, b_lo, b_hi
// +15: 0x0D, 0x10, 0x04, t_lo, t_hi, h_lo, h_hi
uint8_t     data_id; 	// = 0x0A or 0x0D
uint8_t     nx10; 		// = 0x10
union {
	struct {
		uint8_t		len1; // = 0x01
		uint8_t		battery_level; // 0..100 %
		uint8_t		len2; // = 0x02
		uint16_t	battery_mv;
	}t0a;
	struct {
		uint8_t		len; // = 0x04
		int16_t		temperature; // x0.1 C
		uint16_t	humidity; // x0.1 %
	}t0d;
};
*/

func atcCustomParse(d []byte) *DevData {
	var ret = new(DevData)

	rawTemp := int16(binary.LittleEndian.Uint16(d[6:8]))
	rawHumi := binary.LittleEndian.Uint16(d[8:10])
	rawBVolt := binary.LittleEndian.Uint16(d[10:12])
	rawBPercent := uint8(d[12])
	rawCount := uint8(d[13])

	ret.T = float64(rawTemp) / 100.0
	ret.H = float64(rawHumi) / 100.0
	ret.BattV = float64(rawBVolt) / 1000.0
	ret.BattL = uint16(rawBPercent)
	ret.Count = uint32(rawCount)

	return ret
}

func atcParser(_ []byte, sd []ble.ServiceData) (*DevData, error) {
	if len(sd) == 0 {
		return nil, errors.New("ATC: ServiceData is empty")
	}

	for s := range sd {
		switch binary.LittleEndian.Uint16(sd[s].UUID) {
		case UUID_ATC:
			if len(sd[s].Data) == 15 {
				return atcCustomParse(sd[s].Data), nil
			} else {
				return nil, errors.New("ATC: not implemented")
			}
		case UUID_XIAOMI:
			return nil, errors.New("ATC: not implemented")
		}
	}

	return nil, errors.New("ATC: device data not found")
}

func inodeParser(md []byte, _ []ble.ServiceData) (*DevData, error) {
	if len(md) == 0 {
		return nil, errors.New("inode: ManufacturerData is empty")
	}

	rawBattery := binary.LittleEndian.Uint16(md[2:4])
	battery := (rawBattery >> 12) & 0xff
	if battery == 1 {
		battery = 100
	} else {
		battery = 10 * (min(battery, 11) - 1)
	}
	batteryVoltage := (float64(battery)-10)*1.2/100 + 1.8

	rawTemp := binary.LittleEndian.Uint16(md[8:10])
	T := (175.72 * float64(rawTemp) * 4.0 / 65536) - 46.85
	if T < -30 {
		T = -30.0
	} else if T > 70 {
		T = 70.0
	}

	rawPressure := binary.LittleEndian.Uint16(md[6:8])
	P := float64(rawPressure) / 16.0

	rawHumidity := binary.LittleEndian.Uint16(md[10:12])
	H := (125 * float64(rawHumidity) * 4.0 / 65536) - 6.0
	if H < 1 {
		H = 1.0
	} else if H > 100 {
		H = 100
	}

	uptime := uint32(binary.LittleEndian.Uint16(md[12:14]))<<16 | uint32(binary.LittleEndian.Uint16(md[14:16]))

	ret := &DevData{
		T:     T,
		P:     P,
		H:     H,
		Count: uptime,
		BattL: battery,
		BattV: batteryVoltage,
	}

	return ret, nil
}

func DeviceParse(t string, md []byte, sd []ble.ServiceData) (*DevData, error) {
	f, ok := parsers[t]

	if !ok {
		return nil, errors.New("parser not implemented")
	}

	return f(md, sd)
}
