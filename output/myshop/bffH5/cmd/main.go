package main

import (
	"flag"
	"fmt"
	"log"

	"myshop/pkg/config"
	"myshop/bffH5/internal/router"
)

var confPath string

func init() {
	flag.StringVar(&confPath, "config", "configs/config.yaml", "config file")
}

func main() {
	flag.Parse()
	cfg, err := config.Load(confPath)
	if err != nil {
		log.Fatal(err)
	}
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("BFF starting on %s", addr)
	router.NewRouter().Run(addr)
}
