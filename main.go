package main

import (
	"os"
)

func main() {
	cfg, err := setupConfig()
	if err != nil {
		if err == ErrGotHelp {
			os.Exit(1)
		}
		os.Exit(2)
	}
	l := setupLog()
	r := setupRouter(cfg, l)
	r.Run(cfg.Addr)
}
