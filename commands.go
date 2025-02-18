package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/carsondecker/gator/internal/database"
	"github.com/google/uuid"
)

type command struct {
	name string
	args []string
}

type commands struct {
	cmds map[string]func(*state, command) error
}

func initCommands() (*commands, error) {
	cmds := &commands{
		cmds: make(map[string]func(*state, command) error),
	}

	err := cmds.register("login", handlerLogin)
	if err != nil {
		return nil, err
	}

	err = cmds.register("register", handlerRegister)
	if err != nil {
		return nil, err
	}

	err = cmds.register("reset", handlerReset)
	if err != nil {
		return nil, err
	}

	err = cmds.register("users", handlerUsers)
	if err != nil {
		return nil, err
	}

	err = cmds.register("agg", handlerAgg)
	if err != nil {
		return nil, err
	}

	return cmds, nil
}

func (c *commands) register(name string, f func(*state, command) error) error {
	if _, ok := c.cmds[name]; ok {
		return errors.New("command already exists")
	}

	c.cmds[name] = f
	return nil
}

func (c *commands) run(s *state, cmd command) error {
	if _, ok := c.cmds[cmd.name]; !ok {
		return errors.New("command does not exist")
	}

	return c.cmds[cmd.name](s, cmd)
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("login command expects a username argument")
	}

	if _, err := s.db.GetUser(context.Background(), cmd.args[0]); err != nil {
		return errors.New("user doesn't exists")
	}

	err := s.config.SetUser(cmd.args[0])
	if err != nil {
		return err
	}

	fmt.Printf("username has been updated to %s\n", cmd.args[0])
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("register command expects a username argument")
	}

	if _, err := s.db.GetUser(context.Background(), cmd.args[0]); err == nil {
		return errors.New("user already exists")
	}

	_, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
	})
	if err != nil {
		return err
	}

	fmt.Printf("user %s created\n", cmd.args[0])

	err = s.config.SetUser(cmd.args[0])
	if err != nil {
		return err
	}

	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.ResetUsers(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, user := range users {
		if user.Name == s.config.CurrentUserName {
			fmt.Printf("%s (current)\n", user.Name)
		} else {
			fmt.Println(user.Name)
		}
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	feed, err := fetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}
	fmt.Println(feed)
	return nil
}
