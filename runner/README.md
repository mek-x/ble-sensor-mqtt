Simple wrapper to use the configuration from env variables.

Dumps device config from env to devices.yml for usage by ble-sensor-mqtt.

## Supported variables

- `BLE_MQTT_URL` - url of mqtt host, e.g. ssl://mqtt.host.com:8883
- `BLE_MQTT_USER` - user for mqtt host auth
- `BLE_MQTT_PASS` - pass for mqtt host auth
- `BLE_DEVICE_#` - device to add to config file, format is `mac,type,name`, e.g. `aa:bb:cc:dd:ee:ff,ATC,example`
