package main

import (
	"log"
	"os"
)

func main() {
	cfgPath := "/opt/skygate/dashboard.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	_ = cfg
	log.Println("dashboard-daemon: placeholder main (not yet implemented)")
}
