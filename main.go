package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/carsondecker/gator/internal/config"
)

type state struct {
	config *config.Config
}

func main() {
	cfg, err := config.Read()
	handleError(err)

	state := &state{
		config: &cfg,
	}

	commands, err := initCommands()
	handleError(err)

	args := os.Args
	if len(args) < 2 {
		handleError(errors.New("command name required"))
	}

	command := command{
		name: args[1],
		args: args[2:],
	}

	err = commands.run(state, command)
	handleError(err)
}

func handleError(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
