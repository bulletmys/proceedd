package main

import (
	"flag"
	"fmt"
	"github.com/bulletmys/proceedd/client/kv"
	"github.com/golobby/config/v2"
	"github.com/golobby/config/v2/feeder"
	"log"
)

var configPath = flag.String("c", "", "path to config")

func main() {
	flag.Parse()

	c, err := config.New(&feeder.Yaml{Path: * configPath})
	if err != nil {
		log.Fatalf("Error while config parse: %v\n", err)
	}

	if err := kv.Start(c); err != nil {
		log.Fatalf("Failed to start kv module: %v\n", err)
	}

	fmt.Scanln()
}
