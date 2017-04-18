package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	LoadDefaults()
	if store.Mqtt.Host != "iot.eclipse.org" {
		t.Error("LoadDefaults should set the mqtt host to 'iot.eclipse.org': got ", store.Mqtt.Host)
	}
	if store.Mqtt.Port != 1883 {
		t.Error("LoadDefaults should set the mqtt port to 1883: got ", store.Mqtt.Port)
	}
	if !store.Mqtt.Ssl {
		t.Error("LoadDefaults should enable SSL: got SSL disabled")
	}
}
func TestGet(t *testing.T) {
	store = Format{
		Mqtt: MQTTFormat{
			Host:     "mock",
			Port:     1883,
			Ssl:      false,
			Ssl_Auth: false,
		},
	}
	currentConfig := Get()
	if currentConfig.Mqtt.Host != "mock" {
		t.Error("Get() should return the current configuration")
	}
}
