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

func floatToString(i float64) string {
	str := strconv.FormatFloat(i, 'f', 2, 64)
	return str
}

func main() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	config.LoadDefaults()
	homieClient := homie.NewClient("devices/", "172.20.0.100", 1883, false, false, "weatherStation")
	radioClient := radio.NewClient(100, 1, func(metric radio.Metric) {
		nodes := homieClient.Nodes()
		strNodeId := strconv.Itoa(metric.SensorID)
		node, found := nodes[strNodeId]
		if !found {
			homieClient.AddNode(strNodeId, "weather_sensor")
			node = nodes[strNodeId]
		}
		if metric.Temperature != 0 {
			node.Set("temperature", floatToString(metric.Temperature))
		}
		if metric.Humidity != 0 {
			node.Set("humidity", floatToString(metric.Humidity))
		}
		if metric.Pressure != 0 {
			node.Set("pressure", floatToString(metric.Pressure))
		}
		if metric.Battery != 0 {
			node.Set("battery", floatToString(metric.Battery))
		}
	})
	homieClient.Start()
	radioClient.Start("azertyuiopqsdfgh", "433")
	fmt.Println("Subsystems booted")
	select {
	case <-sigc:
		fmt.Println("received interrupt - aborting operations")
		homieClient.Stop()
		radioClient.Stop()
		break
	}
	fmt.Println("main process finished")
}
