package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/database"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatalln(err.Error())
	}

	pg, err := database.NewPostgres(ctx, &cfg.Postgres)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer pg.Close()

	rdb, err := database.NewRedis(ctx, &cfg.Redis)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer rdb.Close()

	fmt.Println(cfg)
}
