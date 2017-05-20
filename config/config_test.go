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
