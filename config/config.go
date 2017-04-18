package config

/*
 {
   "mqtt": {
     "host": "192.0.2.1",
     "port": 1883,
     "ssl": true,
     "ssl_auth": true
   }
 }
*/
type MQTTFormat struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Ssl      bool   `json:"ssl"`
	Ssl_Auth bool   `json:"ssl_auth"`
}
type Format struct {
	Mqtt MQTTFormat `json:"mqtt"`
}

var store Format

func LoadDefaults() {
	store = Format{
		Mqtt: MQTTFormat{
			Host:     "iot.eclipse.org",
			Port:     1883,
			Ssl:      true,
			Ssl_Auth: false,
		},
	}
}

func init() {
	LoadDefaults()
}
func Get() Format {
	return store
}
