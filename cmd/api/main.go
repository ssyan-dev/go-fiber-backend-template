package main

import (
	"fmt"
	"log"

	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
)

func main() {
	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	fmt.Println(cfg)
}