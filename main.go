package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
	"github.com/go-ble/ble/linux/hci/cmd"

	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

const ver = "0.3.0"

var (
	devFile     = flag.String("dev", "devices.yml", "ble devices yaml file")
	scanType    = flag.Bool("as", false, "acitve scan")
	url         = flag.String("url", "", "mqtt host url, e.g. ssl://host.com:8883")
	user        = flag.String("user", "", "mqtt user name")
	pass        = flag.String("pass", "", "mqtt password")
	verbose     = flag.Bool("V", false, "print broadcasted messages")
	topicPrefix = flag.String("topicPre", "/ble-sensor", "topic prefix. Full topic will be {topicPre}/{deviceName}")
)

/* devices.yml example:
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
type devices struct {
	Devices map[string]struct {
		Type string
		Name string
	}
}

var dev devices

type payload struct {
	Time    string `json:"time"`
	Epoch   int64  `json:"timestamp"`
	RSSI    int    `json:"RSSI"`
	Name    string `json:"name"`
	Address string `json:"address"`
	DevData
}

func main() {
	flag.Parse()

	yamlFile, err := ioutil.ReadFile(*devFile)
	if err != nil {
		log.Printf("error, can't read devices file: %v ", err)
	}

	err = yaml.Unmarshal(yamlFile, &dev)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf("ble-sensor-mqtt v%s. Scanning for devices:\n", ver)
	for k := range dev.Devices {
		fmt.Printf("%s: %s - %s\n", k, dev.Devices[k].Name, dev.Devices[k].Type)
	}
	fmt.Printf("\n")

	scanParams := cmd.LESetScanParameters{
		LEScanType:           0x00,   // 0x00: passive, 0x01: active
		LEScanInterval:       0x0004, // 0x0004 - 0x4000; N * 0.625msec
		LEScanWindow:         0x0004, // 0x0004 - 0x4000; N * 0.625msec
		OwnAddressType:       0x00,   // 0x00: public, 0x01: random
		ScanningFilterPolicy: 0x00,   // 0x00: accept all, 0x01: ignore non-white-listed.
	}

	if *scanType {
		scanParams.LEScanType = 0x01
	}

	if len(*url) > 0 {
		establishMqtt(*url, *user, *pass)
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
	d, ok := dev.Devices[a.Addr().String()]
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

	if *verbose {
		log.Printf("%s [%ddBm]: name = %s, type = %s, B = %d%% (%.1fV),"+
			"T = %.3fC, P = %.2fhPa, H = %.1f%%, U = %d\n",
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

	topic := *topicPrefix + "/" + d.Name

	publish(string(payload), topic)
}

func chkErr(err error) {
	switch errors.Cause(err) {
	case nil:
	case context.DeadlineExceeded:
		fmt.Printf("done\n")
	case context.Canceled:
		fmt.Printf("canceled\n")
	default:
		log.Fatalf(err.Error())
	}
}
