package main

import (
	"fmt"
	"log"

	"github.com/carsondecker/gator/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	err = cfg.SetUser("carson")
	if err != nil {
		log.Fatal(err)
	}

	cfg, err = config.Read()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Database URL: %s, Current User: %s\n", cfg.DbURL, cfg.CurrentUserName)
}
