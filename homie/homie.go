package homie

import (
	"errors"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
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
	AddNode(name string, nodeType string, properties []string, settables []SettableProperty)
	Nodes() map[string]Node
}

type SettableProperty struct {
	Name     string
	Callback func(payload string)
}

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

func (homieClient *client) publish(subtopic string, payload string) string {
	id := uuid.New()
	homieClient.publishChan <- stateMessage{subtopic: subtopic, payload: payload, Uuid: id}
	fmt.Println("publication id", id, "submitted")
	return id.String()
}

func (homieClient *client) subscribe(subtopic string, callback func(path string, payload string)) string {
	id := uuid.New()
	homieClient.subscribeChan <- subscribeMessage{subtopic: subtopic, callback: callback, Uuid: id}
	fmt.Println("subscription id", id, "submitted")
	return id.String()
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
	homieClient.publish("$Name", homieClient.FirmwareName())
	homieClient.publish("$mac", homieClient.Mac())
	homieClient.publish("$stats/interval", "10")
	homieClient.publish("$localip", homieClient.Ip())
	homieClient.publish("$fw/Name", homieClient.FirmwareName())
	homieClient.publish("$fw/version", "0.0.1")
	homieClient.publish("implementation", "vx-go-homie")

	// $online must be sent last
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
	fmt.Println("MQTT subsystem started")
	for run {
		select {
		case msg := <-homieClient.publishChan:
			topic := homieClient.getDevicePrefix() + msg.subtopic
			homieClient.mqttClient.Publish(topic, 1, true, msg.payload)
			fmt.Println("publication id", msg.Uuid.String(), "processed")
			break
		case msg := <-homieClient.subscribeChan:
			topic := homieClient.getDevicePrefix() + msg.subtopic
			homieClient.mqttClient.Subscribe(topic, 1, func(mqttClient mqtt.Client, mqttMessage mqtt.Message) {
				msg.callback(mqttMessage.Topic(), string(mqttMessage.Payload()))
			})
			fmt.Println("subscription id", msg.Uuid, "processed")
			break
		case <-homieClient.stopChan:
			run = false
			break
		case <-time.After(10 * time.Second):
			homieClient.publishStats()
			break
		}
	}
	homieClient.mqttClient.Publish(homieClient.getDevicePrefix()+"$online", 1, true, "false")
	homieClient.mqttClient.Disconnect(1000)
	homieClient.stopStatusChan <- true
}

func (homieClient *client) publishStats() {
	homieClient.publish("$stats/uptime", strconv.Itoa(int(time.Since(homieClient.bootTime).Seconds())))
}
func (homieClient *client) Stop() error {
	fmt.Print("stopping mqtt subsystem... ")
	homieClient.stopChan <- true
	for {
		select {
		case <-homieClient.stopStatusChan:
			fmt.Println("done")
			return nil
			break
		case <-time.After(1 * time.Second):
			fmt.Print(".")
			break
		}
	}
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

func (homieClient *client) AddNode(name string, nodeType string, properties []string, settables []SettableProperty) {
	homieClient.nodes[name] = NewNode(
		name, nodeType, properties,
		func(property string, value string) {
			homieClient.publish(name+"/"+property, value)
		})
	homieClient.publish(name+"/$type", nodeType)
	propertyCsv := strings.Join(homieClient.nodes[name].Properties(), ",")
	settablesList := []string{}
	for _, property := range settables {
		fmt.Println("Subscribing for settable properties notifications: ", property.Name)
		homieClient.subscribe(name+"/"+property.Name+"/set", func(path string, payload string) {
			fmt.Println("Settable property update (from path", path, "):", property.Name, " -> ", payload)
			homieClient.nodes[name].Set(property.Name, payload)
			property.Callback(payload)
		})
		settablesList = append(settablesList, property.Name+":settable")
	}
	if len(settablesList) > 0 {
		settablesCsv := strings.Join(settablesList, ",")
		propertyCsv = propertyCsv + "," + settablesCsv
	}
	homieClient.publish(name+"/$properties", propertyCsv)

}

func (homieClient *client) Nodes() map[string]Node {
	return homieClient.nodes
}
