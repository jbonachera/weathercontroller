package radio

import (
	"encoding/json"
	"fmt"
	"github.com/jbonachera/rfm69"
)

type Metric struct {
	Battery     float32 `json:"battery,omitempty"`
	Temperature float32 `json:"temperature,omitempty"`
	Humidity    float32 `json:"humidity,omitempty"`
	Pressure    float32 `json:"pressure,omitempty"`
	SensorID    int     `json:"sensor_id,omitempty"`
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
	callback  func(metric Metric)
}

func NewClient(networkId int, clientId int, callback func(metric Metric)) Client {
	newClient := &client{rfm: nil, networkId: networkId, clientId: clientId, running: false, callback: callback}
	return newClient
}

func (c client) Start(encryptionKey string, frequency string) error {
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
func (c client) Stop() error {
	c.running = false
	return c.rfm.Close()
}

func (c client) loop() {
	rx := make(chan *rfm69.Data, 5)

	c.rfm.OnReceive = func(d *rfm69.Data) {
		rx <- d
	}
	c.running = true
	for c.running {
		select {
		case data := <-rx:
			if data.ToAddress != 255 && data.RequestAck {
				c.rfm.Send(data.ToAck())
			}
			sensorId := int(data.Data[1])
			userData := data.Data[6 : len(data.Data)-1]
			var payload Metric = Metric{}
			err := json.Unmarshal(userData, &payload)
			if err != nil {
				fmt.Println(err)
			} else {
				payload.SensorID = sensorId
				c.callback(payload)
			}
		}

	}
}
