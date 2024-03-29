package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
	"github.com/go-ble/ble/linux/hci/cmd"

	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

var ver = "dev"

/*
	devices.yml example:

```yaml
devices:

	  "01:02:03:04:05:06":
	    type: ATC
	    name: room
	  "02:03:04:05:06:07":
	    type: inode
		name: second_room

```
*/

type device struct {
	Type string
	Name string
}

type config struct {
	Devices map[string]device
	Options map[string]interface{}
}

type payload struct {
	Time    string `json:"time"`
	Epoch   int64  `json:"timestamp"`
	RSSI    int    `json:"RSSI"`
	Name    string `json:"name"`
	Address string `json:"address"`
	DevData
}

var cfg config

func (c *config) updateFromEnv() {
	env := os.Environ()

	for _, e := range env {
		pair := strings.SplitN(e, "=", 2)
		switch {
		case strings.HasPrefix(pair[0], "BLE_DEVICE_"):
			entry := strings.SplitN(pair[1], ",", 3)
			if len(entry) != 3 {
				continue
			}
			c.Devices[entry[0]] = device{Type: entry[1], Name: entry[2]}
		case pair[0] == "BLE_MQTT_URL":
			c.Options["url"] = pair[1]
		case pair[0] == "BLE_MQTT_USER":
			c.Options["user"] = pair[1]
		case pair[0] == "BLE_MQTT_PASS":
			c.Options["pass"] = pair[1]
		case pair[0] == "BLE_MQTT_PFX":
			c.Options["topicPrefix"] = pair[1]
		}
	}
}

func main() {
	log.Printf("ble-sensor-mqtt v. %s", ver)

	cfg.Devices = make(map[string]device)
	cfg.Options = make(map[string]interface{})
	cfg.Options["cfgFile"] = *flag.String("c", "ble-sensor-mqtt.yml", "config file (yaml format)")
	cfg.Options["activeScan"] = *flag.Bool("as", false, "acitve scan")
	cfg.Options["url"] = *flag.String("url", "", "mqtt host url, e.g. ssl://host.com:8883")
	cfg.Options["user"] = *flag.String("user", "", "mqtt user name")
	cfg.Options["pass"] = *flag.String("pass", "", "mqtt password")
	cfg.Options["verbose"] = *flag.Bool("V", false, "print broadcasted messages")
	cfg.Options["topicPrefix"] = *flag.String("pfx", "/ble-sensor", "topic prefix. Full topic will be {topicPre}/{deviceName}")

	flag.Parse()

	if _, err := os.Stat(cfg.Options["cfgFile"].(string)); err == nil {
		yamlFile, err := os.ReadFile(cfg.Options["cfgFile"].(string))
		if err != nil {
			log.Fatalf("error, can't read devices file: %v ", err)
		}

		err = yaml.Unmarshal(yamlFile, &cfg)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	}

	cfg.updateFromEnv()

	if len(cfg.Devices) == 0 {
		log.Fatalf("no devices configured. Stopping...")
	}

	log.Print("scanning for devices:")
	for k := range cfg.Devices {
		log.Printf("%s: %s - %s", k, cfg.Devices[k].Name, cfg.Devices[k].Type)
	}

	scanParams := cmd.LESetScanParameters{
		LEScanType:           0x00,   // 0x00: passive, 0x01: active
		LEScanInterval:       0x0004, // 0x0004 - 0x4000; N * 0.625msec
		LEScanWindow:         0x0004, // 0x0004 - 0x4000; N * 0.625msec
		OwnAddressType:       0x00,   // 0x00: public, 0x01: random
		ScanningFilterPolicy: 0x00,   // 0x00: accept all, 0x01: ignore non-white-listed.
	}

	if cfg.Options["activeScan"].(bool) {
		scanParams.LEScanType = 0x01
	}

	if len(cfg.Options["url"].(string)) > 0 {
		establishMqtt(
			cfg.Options["url"].(string),
			cfg.Options["user"].(string),
			cfg.Options["pass"].(string),
		)

	}

	d, err := linux.NewDevice(ble.OptScanParams(scanParams))
	if err != nil {
		log.Fatalf("can't get device : %s", err)
	}
	ble.SetDefaultDevice(d)

	// Scan for specified durantion, or until interrupted by user.
	ctx := ble.WithSigHandler(context.WithCancel(context.Background()))
	chkErr(ble.Scan(ctx, true, advHandler, nil))
}

func advHandler(a ble.Advertisement) {
	d, ok := cfg.Devices[a.Addr().String()]
	if !ok {
		return
	}

	t := time.Now()

	data, e := DeviceParse(d.Type, a.ManufacturerData(), a.ServiceData())
	if e != nil {
		log.Printf("err: %v", e)
		return
	}

	msg := &payload{
		Time:    t.Format("2006-01-02 15:04:05"),
		Epoch:   t.Unix(),
		RSSI:    a.RSSI(),
		Name:    d.Name,
		Address: a.Addr().String(),
		DevData: *data,
	}

	if cfg.Options["verbose"].(bool) {
		log.Printf("%s [%ddBm]: name = %s, type = %s, B = %d%% (%.1fV),"+
			"T = %.3fC, P = %.2fhPa, H = %.1f%%, U = %d",
			a.Addr().String(),
			a.RSSI(),
			d.Name,
			d.Type,
			msg.BattL,
			msg.BattV,
			msg.T,
			msg.P,
			msg.H,
			msg.Count)
	}

	payload, _ := json.Marshal(msg)

	topic := cfg.Options["topicPrefix"].(string) + "/" + d.Name

	publish(string(payload), topic)
}

func chkErr(err error) {
	switch errors.Cause(err) {
	case nil:
	case context.DeadlineExceeded:
		log.Printf("done")
	case context.Canceled:
		log.Printf("canceled")
	default:
		log.Fatalf(err.Error())
	}
}
