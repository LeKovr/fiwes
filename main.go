package main

import (
	"log"
	"os"
)

// Actual version value will be set at build time
var version = "0.0-dev"

func main() {
	log.Printf("imgserv v %s. Image server", version)
	cfg, err := setupConfig()
	if err != nil {
		if err == ErrGotHelp {
			os.Exit(1)
		}
		os.Exit(2)
	}
	l := setupLog()
	r := setupRouter(cfg, l)
	err = r.Run(cfg.Addr)
	if err != nil {
		log.Fatalf("Run error: %s", err.Error())
	}
}
