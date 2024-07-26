package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func ConnectToMQTT(m model) tea.Cmd {
	return func() tea.Msg {
		clientOps := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s", m.clientUri)).SetClientID("mqtt-tui")
		client := mqtt.NewClient(clientOps)
		isConnected := false
		for !isConnected {
			token := client.Connect()
			if token.Error() != nil {
				return token.Error()
			}
			if client.IsConnected() {
				isConnected = true
			} else {
				time.Sleep(1 * time.Second)
			}
		}
		for _, topic := range m.subscribedTopics {
			client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
				m.sub <- mqttMessage{topic: msg.Topic(), payload: string(msg.Payload())}
			})
		}
		return client
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
	sub              chan mqttMessage // where we'll receive activity notifications
	client           mqtt.Client
	clientUri        string
	subscribedTopics []string
	messageOrder     []string
	messages         map[string]string
	quitting         bool
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		waitForActivity(m.sub), // wait for activity
		ConnectToMQTT(m),       // connect to MQTT
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
			sort.Strings(m.messageOrder)
		}
		m.messages[val.topic] = val.payload
		return m, waitForActivity(m.sub) // wait for next event
	case mqtt.Client:
		m.client = val
		return m, nil
	case error:
		fmt.Printf("error: %v", val)
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m model) View() string {
	if m.client == nil || !m.client.IsConnected() {
		return "Connecting to MQTT..."
	}
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
	broker := os.Args[1]
	if broker == "" {
		fmt.Println("please provide a broker address")
		os.Exit(1)
	}
	topics := os.Args[2:]
	if len(topics) == 0 {
		fmt.Println("please provide at least one topic")
		os.Exit(1)
	}

	subChan := make(chan mqttMessage)
	p := tea.NewProgram(model{
		sub:              subChan,
		clientUri:        broker,
		subscribedTopics: topics,
		messages:         make(map[string]string),
		messageOrder:     make([]string, 0),
	})

	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
