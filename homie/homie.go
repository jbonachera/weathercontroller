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
	bootTime       time.Time
	mqttClient     mqtt.Client
}

func findMacAndIP(ifs []net.Interface) (string, string, error) {
	for _, v := range ifs {
		if v.Flags&net.FlagLoopback != net.FlagLoopback && v.Flags&net.FlagUp == net.FlagUp {
			h := v.HardwareAddr.String()
			if len(h) == 0 {
				continue
			} else {
				addresses, _ := v.Addrs()
				return h, addresses[0].String(), nil
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
	return &client{prefix: prefix, url: url, bootTime: time.Now(), firmwareName: firmwareName}

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
	topic := homieClient.getDevicePrefix() + subtopic
	homieClient.mqttClient.Publish(topic, 1, true, payload)
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
	go homieClient.loop()
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
