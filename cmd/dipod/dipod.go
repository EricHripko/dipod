package main

import (
	"os"

	"github.com/EricHripko/dipod"
	log "github.com/sirupsen/logrus"
)

func main() {
	// setup logging
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)

	// connect to podman
	dipod.Connect()
	// start docker proxy
	dipod.Serve()
}
