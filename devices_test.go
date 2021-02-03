package main

import (
	"testing"

	"github.com/go-ble/ble"
)

func TestParseATC(t *testing.T) {
	sd := []ble.ServiceData{
		{
			UUID: []byte{0x1a, 0x18},
			Data: []byte{27, 50, 60, 56, 193, 164, 4, 9, 139, 13, 168, 11, 87, 23, 4},
		},
	}

	d, e := DeviceParse("ATC", nil, sd)
	if e != nil {
		t.Errorf("Device parse returned an error: %v", e)
	}

	if d.T < 23.0 || d.T > 23.1 {
		t.Errorf("Parsing error, should be 23.08, got %f", d.T)
	}
}

func TestParseInode(t *testing.T) {
	md := []byte{16, 157, 1, 160, 8, 4, 232, 62, 158, 18, 61, 42, 21, 0, 250, 221, 164, 97, 151, 156, 40, 148, 51, 248}

	d, e := DeviceParse("inode", md, nil)
	if e != nil {
		t.Errorf("Device parse returned an error: %v", e)
	}

	if d.T < 4.25 || d.T > 4.27 {
		t.Errorf("Parsing error, should be 4.2658154296875, got %f", d.T)
	}
}
