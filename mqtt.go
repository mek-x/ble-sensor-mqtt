package main

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mqttConnection struct {
	c mqtt.Client
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("MQTT received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("MQTT connected to broker")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("MQTT connection lost: %v", err)
}

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randomString(length int) string {
	return stringWithCharset(length, charset)
}

func establishMqtt(url string, user string, pass string) mqttConnection {

	ssl := tls.Config{
		RootCAs: nil,
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(url)
	opts.SetClientID("ble-sensor-mqtt-" + randomString(5))
	opts.SetUsername(user)
	opts.SetPassword(pass)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)

	opts.SetTLSConfig(&ssl)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return mqttConnection{c: client}
}

func (m *mqttConnection) endConnection() {
	if m.c == nil {
		return
	}
	m.c.Disconnect(0)
}

func (m *mqttConnection) publish(payload string, topic string) {
	if m.c == nil {
		return
	}

	if topic == "" {
		return
	}

	if !m.c.IsConnected() {
		return
	}

	m.c.Publish(topic, 0, true, payload)
}
