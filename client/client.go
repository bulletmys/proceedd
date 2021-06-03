package main

import (
	"flag"
	"github.com/bulletmys/proceedd/client/balancer"
	"github.com/bulletmys/proceedd/client/kv"
	"github.com/golobby/config/v2"
	"github.com/golobby/config/v2/feeder"
	"log"
)

func main() {
	configPath := flag.String("c", "", "path to config")
	fullFlag := flag.Bool("full", false, "full")
	balancerFlag := flag.Bool("balancer", false, "balancer")
	kvFlag := flag.Bool("kv", false, "kv")

	flag.Parse()

	c, err := config.New(&feeder.Yaml{Path: * configPath})
	if err != nil {
		log.Fatalf("Error while config parse: %v\n", err)
	}

	switch {
	case *fullFlag:
		go func() {
			log.Fatal(kv.Start(c))
		}()
		fallthrough
	case *balancerFlag:
		balancer.Start(c)
	case *kvFlag:
		log.Fatal(kv.Start(c))
	default:
		log.Fatalf("no type flag provided")
	}
}
