package main

import (
	"fmt"
	"github.com/jbonachera/weathercontroller/config"
	"github.com/jbonachera/weathercontroller/homie"
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
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	config.LoadDefaults()
	homieClient := homie.NewClient("devices/", "172.20.0.100", 1883, false, false, "weatherStation")
	radioClient := radio.NewClient(100, 1, func(sensorId byte, metric radio.Metric) {
		nodes := homieClient.Nodes()
		strNodeId := strconv.Itoa(int(sensorId))
		node, found := nodes[strNodeId]
		if !found {
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
		fmt.Print("Sensor ", sensorId, ":\t")
		fmt.Println(metric.Dump())
		node.Set("temperature", floatToString(metric.Temperature))
		node.Set("humidity", floatToString(metric.Humidity))
		node.Set("pressure", floatToString(metric.Pressure))
		node.Set("battery", floatToString(metric.Battery))
		node.Set("rssi", intToString(metric.RSSI))

	})
	homieClient.Start()
	radioClient.Start("azertyuiopqsdfgh", "433")
	select {
	case <-sigc:
		fmt.Println("received interrupt - aborting operations")
		homieClient.Stop()
		radioClient.Stop()
		break
	}
	fmt.Println("main process finished")
}
