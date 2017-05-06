package radio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/jbonachera/rfm69"
	"github.com/jbonachera/weathercontroller/log"
)

type Metric struct {
	Battery     float32 `json:"battery,omitempty"`
	Temperature float32 `json:"temperature,omitempty"`
	Humidity    float32 `json:"humidity,omitempty"`
	Pressure    float32 `json:"pressure,omitempty"`
	RSSI        int32   `json:"rssi,omitempty"`
	Uptime      int32   `json:"uptime,omitempty"`
}

func (metric *Metric) Dump() string {
	return fmt.Sprintf("Temperature: %f, Humidity: %f, Pressure: %f, Battery: %f, RSSI: %d, Uptime: %d", metric.Temperature, metric.Humidity, metric.Pressure, metric.Battery, metric.RSSI, metric.Uptime)
}

type Client interface {
	Start(encryptionKey string, frequency string) error
	Stop() error
}

type client struct {
	rfm       *rfm69.Device
	networkId int
	clientId  int
	running   bool
	callback  func(sensorId byte, metric Metric)
	stopped   chan bool
	stop      chan bool
}

func NewClient(networkId int, clientId int, callback func(sensorId byte, metric Metric)) Client {
	newClient := &client{rfm: nil, networkId: networkId, clientId: clientId, running: false, callback: callback}
	return newClient
}

func (c *client) Start(encryptionKey string, frequency string) error {
	var err error
	c.rfm, err = rfm69.NewDevice(byte(c.clientId), byte(c.networkId), true)
	if err != nil {
		return err
	}
	err = c.rfm.Encrypt([]byte(encryptionKey))
	if err != nil {
		panic(err)
	}
	c.rfm.SetFrequency(frequency)
	c.rfm.SetMode(rfm69.RF_OPMODE_RECEIVER)
	go c.loop()
	return nil
}
func (c *client) Stop() error {
	log.Info("stopping radio subsystem")
	c.stop <- true
	for {
		select {
		case <-c.stopped:
			log.Info("radio subsystem")
			return nil
		}
	}

}

func (c *client) loop() {
	rx := make(chan *rfm69.Data, 5)
	c.stopped = make(chan bool, 1)
	c.stop = make(chan bool, 1)

	c.rfm.OnReceive = func(d *rfm69.Data) {
		rx <- d
	}
	log.Info("Radio subsystem started")
	c.running = true
	for c.running {
		select {
		case data := <-rx:
			if data.ToAddress != 255 && data.RequestAck {
				log.Debug("ACK sent")
				c.rfm.Send(data.ToAck())
			}
			buf := bytes.NewReader(data.Data)
			var payload Metric = Metric{}
			err := binary.Read(buf, binary.LittleEndian, &payload)
			if err != nil {
				log.Error(err.Error())
			} else {
				c.callback(data.FromAddress, payload)
			}
		case <-c.stop:
			c.running = false
		}

	}
	c.rfm.Close()
	c.stopped <- true
}
