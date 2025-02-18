package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/carsondecker/gator/internal/config"
	"github.com/carsondecker/gator/internal/database"

	_ "github.com/lib/pq"
)

type state struct {
	db     *database.Queries
	config *config.Config
}

func main() {
	cfg, err := config.Read()
	handleError(err)

	db, err := sql.Open("postgres", cfg.DbURL)
	handleError(err)

	dbQueries := database.New(db)

	state := &state{
		db:     dbQueries,
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
