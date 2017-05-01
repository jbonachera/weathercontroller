package main

import (
	"fmt"
	"github.com/jbonachera/weathercontroller/config"
	"github.com/jbonachera/weathercontroller/homie"
	"github.com/jbonachera/weathercontroller/radio"
	"os"
	"os/signal"
)

func main() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	config.LoadDefaults()
	homieClient := homie.NewClient("devices/", "172.20.0.100", 1883, false, false, "weatherStation")
	radioClient := radio.NewClient(100, 1, func(metric radio.Metric) {
		fmt.Println(metric)
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
