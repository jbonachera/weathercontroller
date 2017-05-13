package config

import (
	"encoding/json"
	"errors"
	"github.com/boltdb/bolt"
	"github.com/jbonachera/weathercontroller/log"
	"sync"
)

const (
	datadir = "/var/lib/weathercontroller/config.db"
)

/*
 {
   "mqtt": {
     "host": "192.0.2.1",
     "port": 1883,
     "ssl": true,
     "ssl_auth": true
   },
   "homie": {
     "name:" "weatherController"
    }
 }
*/

type HomieFormat struct {
	Name string `json:"name,omitempty"`
}

type MQTTFormat struct {
	Prefix   string `json:"prefix,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Ssl      bool   `json:"ssl,omitempty"`
	Ssl_Auth bool   `json:"ssl_auth,omitempty"`
}
type Format struct {
	Mqtt       MQTTFormat  `json:"mqtt,omitempty"`
	Homie      HomieFormat `json:"homie,omitempty"`
	sync.Mutex `json:"-"`
}

var store Format = Format{}

func LoadDefaults() {
	log.Debug("loading default configuration")
	store = Format{
		Mqtt: MQTTFormat{
			Prefix:   "devices/",
			Host:     "172.20.0.100",
			Port:     1883,
			Ssl:      false,
			Ssl_Auth: false,
		},
		Homie: HomieFormat{
			Name: "weatherController",
		},
	}
}

func MergeJSONString(payload string) {
	if err := json.Unmarshal([]byte(payload), &store); err != nil {
		log.Error(err)
	}
}

func Dump() string {
	buf, _ := json.Marshal(store)
	return string(buf)
}
func Save() {
	store.Lock()
	defer store.Unlock()
	db, err := bolt.Open(datadir, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("config"))
		buf, err := json.Marshal(store)
		if err != nil {
			return err
		}
		return b.Put([]byte("store"), buf)
	})
	db.Close()
}

func LoadPersisted() {
	store.Lock()
	db, err := bolt.Open(datadir, 0600, nil)
	if err != nil {
		log.Error(err)
		log.Error("fallback to default config")
		store.Unlock()
		LoadDefaults()
		return
	}
	if db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("config"))
		if b != nil {
			v := b.Get([]byte("store"))
			json.Unmarshal(v, &store)
			store.Unlock()
			return nil
		} else {
			log.Warn("no persisted configuration found")
			return errors.New("bucket 'config' not found")
		}
	}) != nil {
		store.Unlock()
		LoadDefaults()
	}
	db.Close()

}

func Ssl() bool {
	return store.Mqtt.Ssl
}
func SslAuth() bool {
	return store.Mqtt.Ssl_Auth
}
func Host() string {
	return store.Mqtt.Host
}
func Port() int {
	return store.Mqtt.Port
}
func Prefix() string {
	return store.Mqtt.Prefix
}
func HomieName() string {
	return store.Homie.Name
}
