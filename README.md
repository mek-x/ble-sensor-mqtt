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
  -c string
        config file (yaml format) (default "ble-sensor-mqtt.yml")
  -pass string
        mqtt password
  -pfx string
        topic prefix. Full topic will be {topicPre}/{deviceName} (default "/ble-sensor")
  -url string
        mqtt host url, e.g. ssl://host.com:8883
  -user string
        mqtt user name
```

## Configuration

Application looks for `ble-sensor-mqtt.yml` file with all the devices configured. Example file is:

```yaml
devices:
  # BLE MAC address of the device
  "01:02:03:04:05:06":
    type: ATC   # can be ATC or inode
    name: room  # human readable name
  "02:03:04:05:06:07":
    type: inode
    name: second_room
options:
  url: ssl://mqtt.broker.com:8883
  user: username
  pass: password
  topicPrefix: /my-topic-prefix
  activeScan: off
  verbose: off
```

Where:

- _type_ - `ATC` or `inode`. Others will be ignored.
- _name_ - friendly name, used in mqtt topic: Full topic is `{topicPrefix}/{name}`.
- _topicPrefix_ - MQTT prefix. Name has to contain only characters supported by MQTT topics.

Alternatively, these options could be overriden by using environment variables.

### Supported environment variables

- `BLE_MQTT_URL` - _url_ of mqtt host, e.g. **ssl://mqtt.host.com:8883**
- `BLE_MQTT_USER` - _user_ for mqtt host auth
- `BLE_MQTT_PASS` - _pass_ for mqtt host auth
- `BLE_MQTT_PFX` - MQTT _topicPrefix_. Full topic will be `{pfx}/{deviceName}`
- `BLE_DEVICE_#` - device to add to config file, format is `mac,type,name`, e.g. `BLE_DEVICE_0=aa:bb:cc:dd:ee:ff,ATC,example`, `#` is a number.

## Building

In the easiest way, just do `go build`. Golang is required (tested on Linux).

Additionally, [ko](https://ko.build/install/) is used to create minimal container with the application. To build your own container:

```sh
# install ko
go install github.com/google/ko@latest

# build image and publish to local docker
VERSION=devel ko build -L
```

Please see the **ko** [documentation](https://ko.build/) for additional options and how to build for other platforms.

## Ready to use docker images

Currently images are being built and deployed automatically to gitlab
registry available [here](https://gitlab.com/mek_x/ble-sensor-mqtt/container_registry).

The images are built in several flavours: multi-platform manifest (latest and without architecture specific tags), for x86, armv6 (e.g. rpi zero), armv7 (e.g. rpi2), arm64 (e.g. rpi4).

In order to use them simply use this command:

```sh
docker pull registry.gitlab.com/mek_x/ble-sensor-mqtt:latest
```
## License

- MIT

## Dependencies

- [go-ble](https://github.com/go-ble/ble)
- [PAHO MQTT](github.com/eclipse/paho.mqtt.golang)
