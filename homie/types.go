package homie

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/jbonachera/weathercontroller/config"
	"github.com/jbonachera/weathercontroller/log"
	"strconv"
	"time"
)

type Client interface {
	Start() error
	Restart() error
	Name() string
	Id() string
	Url() string
	Ip() string
	Prefix() string
	Mac() string
	Stop() error
	FirmwareName() string
	AddConfigCallback(func(config string))
	AddNode(name string, nodeType string, properties []string, settables []SettableProperty)
	Nodes() map[string]Node
	Reconfigure(prefix string, host string, port int, mqttPrefix string, ssl bool, sslAuth config.TLSFormat, deviceName string)
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
type unsubscribeMessage struct {
	Uuid     uuid.UUID
	subtopic string
}

type client struct {
	id              string
	name            string
	ip              string
	prefix          string
	mac             string
	server          string
	port            int
	mqttPrefix      string
	ssl             bool
	ssl_config      config.TLSFormat
	firmwareName    string
	stopChan        chan bool
	stopStatusChan  chan bool
	publishChan     chan stateMessage
	subscribeChan   chan subscribeMessage
	unsubscribeChan chan unsubscribeMessage
	bootTime        time.Time
	mqttClient      mqtt.Client
	nodes           map[string]Node
	configCallbacks []func(config string)
}

func (homieClient *client) Id() string {
	return homieClient.id
}

func (homieClient *client) Prefix() string {
	return homieClient.prefix
}

func (homieClient *client) Url() string {
	url := homieClient.server + ":" + strconv.Itoa(homieClient.port)
	if homieClient.ssl {
		url = "ssl://" + url
	} else {
		url = "tcp://" + url
	}
	url = url + homieClient.mqttPrefix
	return url
}
func (homieClient *client) Mac() string {
	return homieClient.mac
}
func (homieClient *client) Ip() string {
	return homieClient.ip
}
func (homieClient *client) Name() string {
	return homieClient.name
}

func (homieClient *client) FirmwareName() string {
	return homieClient.firmwareName
}
func (homieClient *client) Nodes() map[string]Node {
	return homieClient.nodes
}

func (homieClient *client) AddConfigCallback(callback func(config string)) {
	homieClient.subscribe("$implementation/config/set", func(path string, payload string) {
		callback(payload)
	})
	homieClient.configCallbacks = append(homieClient.configCallbacks, callback)
}

func (homieClient *client) Reconfigure(prefix string, host string, port int, mqttPrefix string, ssl bool, sslConfig config.TLSFormat, deviceName string) {
	homieClient.name = deviceName
	homieClient.mqttPrefix = mqttPrefix
	homieClient.prefix = prefix
	homieClient.server = host
	homieClient.port = port
	homieClient.ssl = ssl
	homieClient.ssl_config = sslConfig
	log.Info("configuration changed: restarting")
	homieClient.Restart()
}
