package main

import (
	"fmt"
	"github.com/jbonachera/weathercontoller/config"
	"github.com/jbonachera/weathercontoller/homie"
	"os"
	"os/signal"
)

func main() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	config.LoadDefaults()
	homieClient := homie.NewClient("devices/", "172.20.0.100", 1883, false, false, "weatherStation")
	homieClient.Start()
	fmt.Println("connected to mqtt broker")
	select {
	case <-sigc:
		fmt.Println("received interrupt - aborting operations")
		homieClient.Stop()
		break
	}
	fmt.Println("main process finished")
}
