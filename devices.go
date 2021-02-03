package main

import (
	"fmt"

	"github.com/go-ble/ble"
	"github.com/pkg/errors"
)

type DevData struct {
	T      float64 `json:"T"`
	H      float64 `json:"H"`
	P      float64 `json:"P"`
	BattL  uint16  `json:"battLvl"`
	BattV  float64 `json:"battVolt"`
	Uptime uint32  `json:"uptime"`
}

type parser func([]byte, []ble.ServiceData) (*DevData, error)

var parsers = map[string]parser{
	"ATC":   atcParser,
	"inode": inodeParser,
}

func atcParser(_ []byte, sd []ble.ServiceData) (*DevData, error) {
	if len(sd) == 0 {
		return nil, errors.New("ATC: ServiceData is empty")
	}

	fmt.Printf("sd: %v\n", sd)

	ret := &DevData{
		T:      0.0,
		P:      0.0,
		H:      0.0,
		Uptime: 0,
		BattL:  0,
		BattV:  0.0,
	}

	return ret, nil
}

func inodeParser(md []byte, _ []ble.ServiceData) (*DevData, error) {
	if len(md) == 0 {
		return nil, errors.New("inode: ManufacturerData is empty")
	}

	fmt.Printf("md: %v\n", md)

	rawBattery := fromBytesToUint16(md[2:4])
	battery := (rawBattery >> 12) & 0xff
	if battery == 1 {
		battery = 100
	} else {
		battery = 10 * (min(battery, 11) - 1)
	}
	batteryVoltage := (float64(battery)-10)*1.2/100 + 1.8

	rawTemp := fromBytesToUint16(md[8:10])
	T := (175.72 * float64(rawTemp) * 4.0 / 65536) - 46.85
	if T < -30 {
		T = -30.0
	} else if T > 70 {
		T = 70.0
	}

	rawPressure := fromBytesToUint16(md[6:8])
	P := float64(rawPressure) / 16.0

	rawHumidity := fromBytesToUint16(md[10:12])
	H := (125 * float64(rawHumidity) * 4.0 / 65536) - 6.0
	if H < 1 {
		H = 1.0
	} else if H > 100 {
		H = 100
	}

	uptime := uint32(fromBytesToUint16(md[12:14]))<<16 | uint32(fromBytesToUint16(md[14:16]))

	ret := &DevData{
		T:      T,
		P:      P,
		H:      H,
		Uptime: uptime,
		BattL:  battery,
		BattV:  batteryVoltage,
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
