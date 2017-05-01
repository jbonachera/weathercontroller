package homie

import (
	"errors"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"net"
	"strconv"
	"strings"
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
	AddNode(name string, nodeType string, properties []string)
	Nodes() map[string]Node
}

type stateMessage struct {
	subtopic string
	payload  string
}
type subscribeMessage struct {
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

func findMacAndIP(ifs []net.Interface) (string, string, error) {
	for _, v := range ifs {
		if v.Flags&net.FlagLoopback != net.FlagLoopback && v.Flags&net.FlagUp == net.FlagUp {
			h := v.HardwareAddr.String()
			if len(h) == 0 {
				continue
			} else {
				addresses, _ := v.Addrs()
				if len(addresses) > 0 {
					return h, addresses[0].String(), nil
				}
			}
		}
	}
	return "", "", errors.New("could not find a valid network interface")

}

func generateHomieID(mac string) string {
	return strings.Replace(mac, ":", "", -1)
}

func NewClient(prefix string, server string, port int, ssl bool, ssl_auth bool, firmwareName string) Client {
	url := server + ":" + strconv.Itoa(port)
	if ssl {
		url = "ssl://" + url
	} else {
		url = "tcp://" + url
	}
	return &client{
		prefix:        prefix,
		url:           url,
		bootTime:      time.Now(),
		firmwareName:  firmwareName,
		nodes:         map[string]Node{},
		publishChan:   make(chan stateMessage, 10),
		subscribeChan: make(chan subscribeMessage, 10),
	}

}
func (homieClient *client) getDevicePrefix() string {
	return homieClient.Prefix() + homieClient.Id() + "/"
}
func (homieClient *client) getMQTTOptions() *mqtt.ClientOptions {
	o := mqtt.NewClientOptions()
	o.AddBroker(homieClient.url)
	o.SetClientID(homieClient.Id())
	o.SetWill(homieClient.getDevicePrefix()+"$online", "false", 1, true)
	o.SetKeepAlive(10 * time.Second)
	o.SetOnConnectHandler(homieClient.onConnectHandler)
	return o
}

func (homieClient *client) publish(subtopic string, payload string) {
	homieClient.publishChan <- stateMessage{subtopic: subtopic, payload: payload}
}

func (homieClient *client) subscribe(subtopic string, callback func(path string, payload string)) {
	homieClient.subscribeChan <- subscribeMessage{subtopic: subtopic, callback: callback}
}

func (homieClient *client) onConnectHandler(client mqtt.Client) {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	mac, ip, err := findMacAndIP(ifaces)
	if err != nil {
		panic(err)
	}
	id := generateHomieID(mac)
	if err != nil {
		panic(err)
	}
	homieClient.ip = ip
	homieClient.mac = mac
	homieClient.id = id
	go homieClient.loop()

	homieClient.publish("$homie", "2.0.0")
	homieClient.publish("$name", homieClient.FirmwareName())
	homieClient.publish("$mac", homieClient.Mac())
	homieClient.publish("$stats/interval", "10")
	homieClient.publish("$localip", homieClient.Ip())
	homieClient.publish("$fw/name", homieClient.FirmwareName())
	homieClient.publish("$fw/version", "0.0.1")
	homieClient.publish("implementation", "vx-go-homie")

	// homieClient must be sent last
	homieClient.publish("$online", "true")
}

func (homieClient *client) Start() error {
	tries := 0
	homieClient.mqttClient = mqtt.NewClient(homieClient.getMQTTOptions())
	for !homieClient.mqttClient.IsConnected() && tries < 10 {
		if token := homieClient.mqttClient.Connect(); token.Wait() && token.Error() != nil {
			time.Sleep(5 * time.Second)
			tries += 1
		} else {
		}
	}
	if tries >= 10 {
		return errors.New("could not connect to MQTT at " + homieClient.Url())
	} else {
		return nil
	}

}

func (homieClient *client) loop() {
	run := true
	homieClient.stopChan = make(chan bool, 1)
	homieClient.stopStatusChan = make(chan bool, 1)
	for run {
		select {
		case msg := <-homieClient.publishChan:
			topic := homieClient.getDevicePrefix() + msg.subtopic
			homieClient.mqttClient.Publish(topic, 1, true, msg.payload)
			break
		case msg := <-homieClient.subscribeChan:
			topic := homieClient.getDevicePrefix() + msg.subtopic
			homieClient.mqttClient.Subscribe(topic, 1, func(mqttClient mqtt.Client, mqttMessage mqtt.Message) {
				msg.callback(mqttMessage.Topic(), string(mqttMessage.Payload()))
			})
			break
		case <-homieClient.stopChan:
			run = false
			break
		case <-time.After(10 * time.Second):
			homieClient.publishStats()
			break
		}
	}
	homieClient.publish("$online", "false")
	homieClient.mqttClient.Disconnect(1000)
	homieClient.stopStatusChan <- true
}

func (homieClient *client) publishStats() {
	homieClient.publish("$stats/uptime", strconv.Itoa(int(time.Since(homieClient.bootTime).Seconds())))
}
func (homieClient *client) Stop() error {
	homieClient.stopChan <- true
	select {
	case <-homieClient.stopStatusChan:
		return nil
		break
	case <-time.After(10 * time.Second):
		break
	}
	return errors.New("MQTT did not stop after 10s")
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

func (homieClient *client) AddNode(name string, nodeType string, properties []string) {
	homieClient.nodes[name] = NewNode(
		name, nodeType, properties,
		func(property string, value string) {
			homieClient.publish(name+"/"+property, value)
		})
	homieClient.publish(name+"/$type", nodeType)
	homieClient.publish(name+"/$properties", strings.Join(homieClient.nodes[name].Properties(), ","))
}

func (homieClient *client) Nodes() map[string]Node {
	return homieClient.nodes
}
