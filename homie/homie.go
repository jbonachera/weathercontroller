package homie

import (
	"errors"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/jbonachera/weathercontroller/log"
	"net"
	"strconv"
	"strings"
	"time"
)

func NewClient(prefix string, server string, port int, ssl bool, ssl_auth bool, firmwareName string) Client {
	return &client{
		prefix:          prefix,
		server:          server,
		port:            port,
		ssl:             ssl,
		ssl_auth:        ssl_auth,
		bootTime:        time.Now(),
		firmwareName:    firmwareName,
		nodes:           map[string]Node{},
		publishChan:     make(chan stateMessage, 10),
		subscribeChan:   make(chan subscribeMessage, 10),
		unsubscribeChan: make(chan unsubscribeMessage, 10),
	}

}
func (homieClient *client) getMQTTOptions() *mqtt.ClientOptions {
	o := mqtt.NewClientOptions()
	o.AddBroker(homieClient.Url())
	o.SetClientID(homieClient.Id())
	o.SetWill(homieClient.getDevicePrefix()+"$online", "false", 1, true)
	o.SetKeepAlive(10 * time.Second)
	o.SetOnConnectHandler(homieClient.onConnectHandler)
	return o
}

func (homieClient *client) publish(subtopic string, payload string) string {
	id := uuid.New()
	homieClient.publishChan <- stateMessage{subtopic: subtopic, payload: payload, Uuid: id}
	log.Trace("publication id", id, "submitted")
	return id.String()
}

func (homieClient *client) unsubscribe(subtopic string) string {
	id := uuid.New()
	homieClient.unsubscribeChan <- unsubscribeMessage{subtopic: subtopic, Uuid: id}
	log.Trace("unsubscription id", id, "submitted")
	return id.String()
}

func (homieClient *client) subscribe(subtopic string, callback func(path string, payload string)) string {
	id := uuid.New()
	homieClient.subscribeChan <- subscribeMessage{subtopic: subtopic, callback: callback, Uuid: id}
	log.Trace("subscription id", id, "submitted")
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
	homieClient.publish("$implementation", "vx-go-homie")

	homieClient.subscribe("$implementation/config/set", func(path string, payload string) {
		homieClient.Restart()
	})

	// $online must be sent last
	homieClient.publish("$online", "true")
}

func (homieClient *client) Start() error {
	tries := 0
	log.Debug("creating mqtt client")
	homieClient.mqttClient = mqtt.NewClient(homieClient.getMQTTOptions())
	homieClient.bootTime = time.Now()
	log.Debug("connecting to mqtt server")
	for !homieClient.mqttClient.IsConnected() && tries < 10 {
		if token := homieClient.mqttClient.Connect(); token.Wait() && token.Error() != nil {
			log.Warn("connection to mqtt server failed. will retry in 5 seconds")
			time.Sleep(5 * time.Second)
			tries += 1
		} else {
			log.Debug("connected to mqtt server")
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
	log.Info("mqtt subsystem started")
	for run {
		select {
		case msg := <-homieClient.publishChan:
			topic := homieClient.getDevicePrefix() + msg.subtopic
			homieClient.mqttClient.Publish(topic, 1, true, msg.payload)
			log.Trace("publication id", msg.Uuid.String(), "processed")
			break
		case msg := <-homieClient.unsubscribeChan:
			topic := homieClient.getDevicePrefix() + msg.subtopic
			homieClient.mqttClient.Unsubscribe(topic)
			log.Trace("unsubscription id", msg.Uuid, "processed")
			break
		case msg := <-homieClient.subscribeChan:
			topic := homieClient.getDevicePrefix() + msg.subtopic
			homieClient.mqttClient.Subscribe(topic, 1, func(mqttClient mqtt.Client, mqttMessage mqtt.Message) {
				msg.callback(mqttMessage.Topic(), string(mqttMessage.Payload()))
			})
			log.Trace("subscription id", msg.Uuid, "processed")
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
	log.Info("stopping mqtt subsystem")
	homieClient.stopChan <- true
	for {
		select {
		case <-homieClient.stopStatusChan:
			log.Info("mqtt subsystem stopped")
			return nil
			break
		}
	}
}

func (homieClient *client) AddNode(name string, nodeType string, properties []string, settables []SettableProperty) {
	homieClient.nodes[name] = NewNode(
		name, nodeType, properties, settables,
		func(property string, value string) {
			homieClient.publish(name+"/"+property, value)
		})
	homieClient.publishNode(homieClient.nodes[name])
}
func (homieClient *client) publishNode(node Node) {
	name := node.Name()
	nodeType := node.Type()
	settables := node.Settables()

	homieClient.publish(name+"/$type", nodeType)

	propertyCsv := strings.Join(homieClient.nodes[name].Properties(), ",")
	settablesList := []string{}
	for _, property := range settables {
		log.Debug("Subscribing for settable properties notifications: ", property.Name)
		prop := property.Name
		homieClient.subscribe(name+"/"+prop+"/set", func(path string, payload string) {
			log.Debug("Settable property update (from path", path, "):", prop, " -> ", payload)
			homieClient.nodes[name].Set(prop, payload)
			property.Callback(payload)
		})
		homieClient.subscribe(name+"/"+prop, func(path string, payload string) {
			log.Debug("restoring old value for property ", prop, ": ", payload)
			homieClient.nodes[name].Set(prop, payload)
			homieClient.unsubscribe(name + "/" + prop)
		})
		settablesList = append(settablesList, property.Name+":settable")
	}
	if len(settablesList) > 0 {
		settablesCsv := strings.Join(settablesList, ",")
		propertyCsv = propertyCsv + "," + settablesCsv
	}
	homieClient.publish(name+"/$properties", propertyCsv)

}

func (homieClient *client) Restart() error {
	log.Info("restarting mqtt subsystem")
	homieClient.Stop()
	homieClient.Start()
	for _, node := range homieClient.Nodes() {
		homieClient.publishNode(node)
	}
	return nil
}
