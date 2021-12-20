# ble-sensor-mqtt

[![pipeline status](https://gitlab.com/mek_x/ble-sensor-mqtt/badges/master/pipeline.svg)](https://gitlab.com/mek_x/ble-sensor-mqtt/-/commits/master)

This project is intended to be simple application used to acquire various sensor data (mainly weather data, i.e. temperature, humidity, pressure)
from Bluetooth (BLE) devices and publish them to configured MQTT broker for further processing.

## Supported devices

Supported and tested devices are:

- [iNode Care Sensor PHT](https://inode.pl/iNode-Care-Sensor-PHT-p34)
- [ATC firmware](https://github.com/pvvx/ATC_MiThermometer) for Xiaomi Miija (LYWSD03MMC) device

## Basis of operation

Application uses on-board bluetooth device (hci0) in scanning mode to listen for advertisement packets from devices.
When packet is received it is parsed and sent to configured MQTT broker (so far only brokers with TLS connectivity are supported).

```
      adv packet                     device present            packet successfully
       received                        in config                     parsed
┌──────┐       ┌───────────────────────┐      ┌─────────────────────┐     ┌─────────────────┐
│      │       │                       │      │                     │     │                 │
│ hci0 ├──────►│  check configuration  ├─────►│ parse device packet ├────►│ publish to MQTT │
│      │       │      for device       │      │                     │     │                 │
└──────┘       └───────────────────────┘      └─────────────────────┘     └─────────────────┘
```

## Usage

```
Usage of ./ble-sensor-mqtt:
  -V    print broadcasted messages
  -as
        acitve scan
  -dev string
        ble devices yaml file (default "devices.yml")
  -pass string
        mqtt password
  -topicPre string
        topic prefix. Full topic will be {topicPre}/{deviceName} (default "/ble-sensor")
  -url string
        mqtt host url, e.g. ssl://host.com:8883
  -user string
        mqtt user name
```

## Configuration

Applications looks for `devices.yml` file with all the devices configured. Example file is:

```yaml
devices:
  "01:02:03:04:05:06":
    type: ATC
    name: room
  "02:03:04:05:06:07":
    type: inode
    name: second_room
```

Where:
- _type_ - `ATC` or `inode`. Others will be ignored.
- _name_ - friendly name, used in mqtt topic: Full topic is `{topicPre}/{name}`. _topicPre_ is configured in parameters. Name has to contain only characters supported by MQTT topics.
- _01:02:03:04:05:06_ - device BLE MAC address

## Building

In the easiest way, just do `go build`. Golang is required (tested on Linux).

Additionally `Dockerfile` is included to create minimal container with the application (ARM build by default) for use on, e.g., 
Raspberry Pi (tested on RPi Zero and RPi 4) deployed with the [balenaOS](https://www.balena.io/os).

## License

- MIT
