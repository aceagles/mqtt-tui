package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func listenForActivity(sub chan mqttMessage) tea.Cmd {
	return func() tea.Msg {
		for {
			time.Sleep(time.Millisecond * time.Duration(rand.Int63n(900)+100)) // nolint:gosec
			sub <- mqttMessage{}
		}
	}
}

type mqttMessage struct {
	topic   string
	payload string
}

// A command that waits for the activity on a channel.
func waitForActivity(sub chan mqttMessage) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

type model struct {
	sub          chan mqttMessage // where we'll receive activity notifications
	client       mqtt.Client
	messageOrder []string
	messages     map[string]string
	quitting     bool
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		waitForActivity(m.sub), // wait for activity
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch val := msg.(type) {
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit
	case mqttMessage:
		if _, ok := m.messages[val.topic]; !ok {
			m.messageOrder = append(m.messageOrder, val.topic)
		}
		m.messages[val.topic] = val.payload
		return m, waitForActivity(m.sub) // wait for next event
	default:
		return m, nil
	}
}

func (m model) View() string {
	var s string
	for _, topic := range m.messageOrder {
		s += fmt.Sprintf("%s: %s\n", topic, m.messages[topic])
	}
	if m.quitting {
		s += "\n"
	}
	return s
}

func main() {

	broker := "tcp://localhost:1883"
	clientOps := mqtt.NewClientOptions().AddBroker(broker).SetClientID("mqtt-tui")
	client := mqtt.NewClient(clientOps)
	token := client.Connect()
	if token.Error() != nil {
		fmt.Println("could not connect to MQTT broker")
		os.Exit(1)
	}
	defer client.Disconnect(250)
	subChan := make(chan mqttMessage)
	p := tea.NewProgram(model{
		sub:          subChan,
		client:       client,
		messages:     make(map[string]string),
		messageOrder: make([]string, 0),
	})
	<-token.Done()

	client.Subscribe("#", 0, func(client mqtt.Client, msg mqtt.Message) {

		subChan <- mqttMessage{topic: msg.Topic(), payload: string(msg.Payload())}
	})

	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
