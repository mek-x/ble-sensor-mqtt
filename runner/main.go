package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"gopkg.in/yaml.v2"
)

type device struct {
	Type string
	Name string
}

func main() {
	var err error

	env := os.Environ()

	devices := make(map[string]device)
	options := make(map[string]string)

	for _, e := range env {
		pair := strings.SplitN(e, "=", 2)
		switch {
		case strings.HasPrefix(pair[0], "BLE_DEVICE_"):
			entry := strings.SplitN(pair[1], ",", 3)
			if len(entry) != 3 {
				continue
			}
			fmt.Println(pair[0], "=", pair[1])
			devices[entry[0]] = device{Type: entry[1], Name: entry[2]}
		case pair[0] == "BLE_MQTT_URL":
			options["url"] = pair[1]
		case pair[0] == "BLE_MQTT_USER":
			options["user"] = pair[1]
		case pair[0] == "BLE_MQTT_PASS":
			options["pass"] = pair[1]
		}

	}

	cfg := make(map[string]interface{})
	cfg["devices"] = devices

	file, err := yaml.Marshal(&cfg)
	if err != nil {
		panic(err)
	}

	os.WriteFile("devices.yml", file, 0644)

	var args []string

	args = append(args, os.Args[1])

	for k, v := range options {
		args = append(args, "-"+k)
		args = append(args, v)
	}

	fmt.Println("Starting:", args)
	err = syscall.Exec(os.Args[1], args, env)
	if err != nil {
		panic(err)
	}
}
