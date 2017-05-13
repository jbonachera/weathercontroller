package config

import (
	"encoding/json"
	"errors"
	"github.com/boltdb/bolt"
	"github.com/jbonachera/weathercontroller/log"
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
	Mqtt  MQTTFormat  `json:"mqtt,omitempty"`
	Homie HomieFormat `json:"homie,omitempty"`
}

var store Format = Format{}
var db *bolt.DB = nil

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

func init() {
	var err error
	db, err = bolt.Open(datadir, 0600, nil)
	if err != nil {
		panic(err)
	}
}

func Stop() {
	db.Close()
}

func Save() {
	defer func() {
		if r := recover(); r != nil {
			log.Fatal("Unknown error occured when saving config")
			log.Fatal(r)
		}
	}()
	log.Debug("updating saved configuration")
	log.Debug("opening database")
	log.Debug("aquiring r/w transaction")
	db.Update(func(tx *bolt.Tx) error {
		log.Debug("transaction aquired")
		b := tx.Bucket([]byte("config"))
		if b == nil {
			var err error
			b, err = tx.CreateBucket([]byte("config"))
			if err != nil {
				return err
			}
		}
		buf, err := json.Marshal(store)
		if err != nil {
			return err
		}
		log.Debug("updating data")
		return b.Put([]byte("store"), buf)
	})
	log.Debug("closing database")
}

func LoadPersisted() {
	if db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("config"))
		if b != nil {
			v := b.Get([]byte("store"))
			json.Unmarshal(v, &store)
			return nil
		} else {
			log.Warn("no persisted configuration found")
			return errors.New("bucket 'config' not found")
		}
	}) != nil {
		LoadDefaults()
	}

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
