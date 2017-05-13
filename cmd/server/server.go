package main

import (
	"github.com/jbonachera/weathercontroller/config"
	"github.com/jbonachera/weathercontroller/homie"
	"github.com/jbonachera/weathercontroller/log"
	"github.com/jbonachera/weathercontroller/radio"
	"os"
	"os/signal"
	"strconv"
)

func floatToString(i float32) string {
	str := strconv.FormatFloat(float64(i), 'f', 2, 64)
	return str
}
func intToString(i int32) string {
	str := strconv.Itoa(int(i))
	return str
}
func main() {
	log.Info("main process starting")
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)
	config.LoadPersisted()
	log.SetLevel(log.DEBUG)
	homieClient := homie.NewClient(config.Prefix(), config.Host(), config.Port(), config.Ssl(), config.SslAuth(), "weatherStation")
	radioClient := radio.NewClient(100, 1, func(sensorId byte, metric radio.Metric) {
		nodes := homieClient.Nodes()
		strNodeId := strconv.Itoa(int(sensorId))
		node, found := nodes[strNodeId]
		if !found {
			log.Info("discovered new sensor: ", sensorId)
			homieClient.AddNode(strNodeId, "weather_sensor",
				[]string{
					"temperature",
					"humidity",
					"pressure",
					"rssi",
					"uptime",
					"battery",
				},
				[]homie.SettableProperty{
					{"room", func(payload string) {}},
					{"fancy_name", func(payload string) {}},
				},
			)
			node = nodes[strNodeId]
		}
		log.Info("Sensor ", sensorId, ": "+metric.Dump())
		node.Set("temperature", floatToString(metric.Temperature))
		node.Set("humidity", floatToString(metric.Humidity))
		node.Set("pressure", floatToString(metric.Pressure))
		node.Set("battery", floatToString(metric.Battery))
		node.Set("rssi", intToString(metric.RSSI))
		node.Set("uptime", intToString(metric.Uptime))

	})
	homieClient.AddConfigCallback(func(payload string) {
		log.Debug("config changeset: ", payload)
		config.MergeJSONString(payload)
		log.Debug("new config: ", config.Dump())
		homieClient.Reconfigure(config.Prefix(), config.Host(), config.Port(), config.Ssl(), config.SslAuth())
	})
	go homieClient.Start()
	go radioClient.Start("azertyuiopqsdfgh", "433")
	select {
	case <-sigc:
		log.Warn("received interrupt - aborting operations")
		homieClient.Stop()
		radioClient.Stop()
		break
	}
	log.Info("main process finished")
}
