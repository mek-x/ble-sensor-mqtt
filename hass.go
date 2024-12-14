package main

import "encoding/json"

type HassCfgDevice struct {
	Identifiers  string `json:"ids"`
	Name         string `json:"name"`
	Manufacturer string `json:"mf,omitempty"`
	Model        string `json:"mdl,omitempty"`
	SwVersion    string `json:"sw,omitempty"`
	HwVersion    string `json:"hw,omitempty"`
	SerialNumber string `json:"sn,omitempty"`
}

type HassCfgOrigin struct {
	Name       string `json:"name"`
	SwVersion  string `json:"sw,omitempty"`
	SupportUrl string `json:"url,omitempty"`
}

type HassCfgComponent struct {
	UniqueId           string `json:"unique_id"`
	Platform           string `json:"p"`
	DeviceClass        string `json:"device_class"`
	UnitOfMeasurement  string `json:"unit_of_measurement"`
	ValueTemplate      string `json:"value_template"`
	EntityCategory     string `json:"entity_category,omitempty"`
	SuggestedPrecision int    `json:"suggested_display_precision,omitempty"`
	/*
	   "p": "sensor",
	   "device_class":"temperature",
	   "unit_of_measurement":"°C",
	   "value_template":"{{ value_json.temperature}}",
	   "unique_id":"temp01ae_t",
	*/
}

type DiscoveryPayload struct {
	Device     HassCfgDevice               `json:"dev"`
	Origin     HassCfgOrigin               `json:"o"`
	Components map[string]HassCfgComponent `json:"cmps"`
	StateTopic string                      `json:"state_topic"`
	QoS        int                         `json:"qos"`
}

const HASS_PFX = "homeassistant/device/"

func HassGetTopic(name string) string {
	return HASS_PFX + name + "/config"
}

func HassGetDiscoveryMessage(name string, devType string, addr string, pfx string) string {
	payload := DiscoveryPayload{
		Device: HassCfgDevice{
			Identifiers:  addr,
			Name:         name,
			Manufacturer: "Mek",
			Model:        devType,
			SwVersion:    ver,
			HwVersion:    "1.0",
			SerialNumber: addr,
		},
		Origin: HassCfgOrigin{
			Name:       "ble-sensor-mqtt",
			SupportUrl: "https://gitlab.com/mek_x/ble-sensor-mqtt",
			SwVersion:  ver,
		},
		Components: map[string]HassCfgComponent{
			"temperature": {
				Platform:           "sensor",
				DeviceClass:        "temperature",
				UnitOfMeasurement:  "°C",
				ValueTemplate:      "{{ value_json.T }}",
				UniqueId:           name + "_temperature",
				SuggestedPrecision: 2,
			},
			"humidity": {
				Platform:           "sensor",
				DeviceClass:        "humidity",
				UnitOfMeasurement:  "%",
				ValueTemplate:      "{{ value_json.H }}",
				UniqueId:           name + "_humidity",
				SuggestedPrecision: 0,
			},
			"battery": {
				Platform:           "sensor",
				DeviceClass:        "battery",
				UnitOfMeasurement:  "%",
				EntityCategory:     "diagnostic",
				ValueTemplate:      "{{ value_json.battLvl }}",
				UniqueId:           name + "_battery",
				SuggestedPrecision: 0,
			},
			"rssi": {
				Platform:           "sensor",
				DeviceClass:        "signal_strength",
				UnitOfMeasurement:  "dBm",
				EntityCategory:     "diagnostic",
				ValueTemplate:      "{{ value_json.RSSI }}",
				UniqueId:           name + "_rssi",
				SuggestedPrecision: 0,
			},
		},
		StateTopic: pfx + "/" + name,
		QoS:        0,
	}

	if devType == "inode" {
		payload.Components["pressure"] = HassCfgComponent{
			Platform:           "sensor",
			DeviceClass:        "pressure",
			UnitOfMeasurement:  "hPa",
			ValueTemplate:      "{{ value_json.P }}",
			UniqueId:           name + "_pressure",
			SuggestedPrecision: 1,
		}
	}

	msg, _ := json.Marshal(&payload)

	return string(msg)
}
