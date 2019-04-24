package main

import (
	"os"
)

func main() {
	cfg, err := setupConfig()
	if err != nil {
		if err.Error() == "ERR1" {
			os.Exit(1)
		}
		os.Exit(2)
	}
	r := setupRouter(cfg)
	r.Run(cfg.Addr)
}
