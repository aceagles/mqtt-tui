package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	broker := os.Getenv("MQTT_BROKER")
	if broker == "" {
		broker = "tcp://localhost:1883"
	}

	clientOps := mqtt.NewClientOptions().AddBroker(broker)
	client := mqtt.NewClient(clientOps)
	client.Connect()
	defer client.Disconnect(250)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			return
		case <-time.After(time.Second):
			client.Publish(fmt.Sprintf("test%d", rand.IntN(6)), 0, false, fmt.Sprintf("Hello, world! - %d", rand.IntN(100)))
		}
	}
}
