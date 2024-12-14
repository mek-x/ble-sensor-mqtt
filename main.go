package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"
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

type Opts struct {
	CfgFile       string
	ActiveScan    bool
	Url           string
	User          string
	Pass          string
	Verbose       bool
	TopicPrefix   string
	Interval      int
	HassDiscovery bool
}

type config struct {
	Devices map[string]device
	Options Opts
	msgPipe chan payload
	mqtt    mqttConnection
}

type payload struct {
	Time    string `json:"time"`
	Epoch   int64  `json:"timestamp"`
	RSSI    int    `json:"RSSI"`
	Name    string `json:"name"`
	Address string `json:"address"`
	DevData
}

type measurement map[string]payload

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
			c.Options.Url = pair[1]
		case pair[0] == "BLE_MQTT_USER":
			c.Options.User = pair[1]
		case pair[0] == "BLE_MQTT_PASS":
			c.Options.Pass = pair[1]
		case pair[0] == "BLE_MQTT_PFX":
			c.Options.TopicPrefix = pair[1]
		case pair[0] == "BLE_MQTT_INTER":
			var err error
			c.Options.Interval, err = strconv.Atoi(pair[1])
			if err != nil {
				c.Options.Interval = 0
			}
		case pair[0] == "BLE_HASS_DSCVR":
			if pair[1] == "true" || pair[1] == "1" || pair[1] == "on" {
				c.Options.HassDiscovery = true
			} else {
				c.Options.HassDiscovery = false
			}
		}
	}
}

func main() {
	log.Printf("ble-sensor-mqtt v. %s", ver)

	cfg.Devices = make(map[string]device)
	flag.StringVar(&cfg.Options.CfgFile, "c", "ble-sensor-mqtt.yml", "config file (yaml format)")
	flag.BoolVar(&cfg.Options.ActiveScan, "as", false, "active scan")
	flag.StringVar(&cfg.Options.Url, "url", "", "mqtt host url, e.g. ssl://host.com:8883")
	flag.StringVar(&cfg.Options.User, "user", "", "mqtt user name")
	flag.StringVar(&cfg.Options.Pass, "pass", "", "mqtt password")
	flag.BoolVar(&cfg.Options.Verbose, "V", false, "print broadcasted messages")
	flag.StringVar(&cfg.Options.TopicPrefix, "pfx", "ble-sensor", "topic prefix. Full topic will be {topicPre}/{deviceName}")
	flag.IntVar(&cfg.Options.Interval, "t", 0, "how often send messages to broker in seconds, 0 - as soon as packet received")
	flag.BoolVar(&cfg.Options.HassDiscovery, "hass", false, "enable Home Assistant MQTT discovery")

	flag.Parse()

	if _, err := os.Stat(cfg.Options.CfgFile); err == nil {
		yamlFile, err := os.ReadFile(cfg.Options.CfgFile)
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

	if cfg.Options.ActiveScan {
		scanParams.LEScanType = 0x01
	}

	if len(cfg.Options.Url) > 0 {
		cfg.mqtt = establishMqtt(
			cfg.Options.Url,
			cfg.Options.User,
			cfg.Options.Pass,
		)
		defer func() {
			log.Println("Disconnect")
			cfg.mqtt.endConnection()
		}()
	}

	if cfg.Options.HassDiscovery {
		for addr, dev := range cfg.Devices {
			topic := HassGetTopic(dev.Name)
			msg := HassGetDiscoveryMessage(dev.Name, dev.Type, addr, cfg.Options.TopicPrefix)

			cfg.mqtt.publish(msg, topic)
		}
	}

	cfg.msgPipe = make(chan payload)
	defer close(cfg.msgPipe)

	d, err := linux.NewDevice(ble.OptScanParams(scanParams))
	if err != nil {
		log.Fatalf("can't get device : %s", err)
	}
	ble.SetDefaultDevice(d)

	go cfg.sender()
	// Scan for specified duration, or until interrupted by user.
	ctx := ble.WithSigHandler(context.WithCancel(context.Background()))
	chkErr(ble.Scan(ctx, true, cfg.advHandler, nil))
}

func (cfg *config) advHandler(a ble.Advertisement) {
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

	msg := payload{
		Time:    t.Format("2006-01-02 15:04:05"),
		Epoch:   t.Unix(),
		RSSI:    a.RSSI(),
		Name:    d.Name,
		Address: a.Addr().String(),
		DevData: *data,
	}

	if cfg.Options.Verbose {
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

	cfg.msgPipe <- msg
}

func (cfg *config) sendPayload(msg payload) {
	payload, _ := json.Marshal(msg)
	topic := cfg.Options.TopicPrefix + "/" + msg.Name

	//log.Println(string(payload))
	cfg.mqtt.publish(string(payload), topic)
}

func (cfg *config) sender() {

	measurements := make(measurement)
	var ticker *time.Ticker
	if cfg.Options.Interval > 0 {
		ticker = time.NewTicker(time.Duration(cfg.Options.Interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg := <-cfg.msgPipe:
				measurements[msg.Address] = msg
			case <-ticker.C:
				for _, msg := range measurements {
					cfg.sendPayload(msg)
				}
			}
		}
	} else {
		for {
			msg := <-cfg.msgPipe
			cfg.sendPayload(msg)
		}
	}
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
