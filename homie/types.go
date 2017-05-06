package homie

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"time"
)

type Client interface {
	Start() error
	Id() string
	Url() string
	Ip() string
	Prefix() string
	Mac() string
	Stop() error
	FirmwareName() string
	AddNode(name string, nodeType string, properties []string, settables []SettableProperty)
	Nodes() map[string]Node
}
type SettableProperty struct {
	Name     string
	Callback func(payload string)
}

// TODO track message processing time
type stateMessage struct {
	Uuid     uuid.UUID
	subtopic string
	payload  string
}
type subscribeMessage struct {
	Uuid     uuid.UUID
	subtopic string
	callback func(path string, payload string)
}
type client struct {
	id             string
	ip             string
	prefix         string
	mac            string
	url            string
	firmwareName   string
	stopChan       chan bool
	stopStatusChan chan bool
	publishChan    chan stateMessage
	subscribeChan  chan subscribeMessage
	bootTime       time.Time
	mqttClient     mqtt.Client
	nodes          map[string]Node
}

func (homieClient *client) Id() string {
	return homieClient.id
}

func (homieClient *client) Prefix() string {
	return homieClient.prefix
}

func (homieClient *client) Url() string {
	return homieClient.url
}
func (homieClient *client) Mac() string {
	return homieClient.mac
}
func (homieClient *client) Ip() string {
	return homieClient.ip
}
func (homieClient *client) FirmwareName() string {
	return homieClient.firmwareName
}
func (homieClient *client) Nodes() map[string]Node {
	return homieClient.nodes
}
