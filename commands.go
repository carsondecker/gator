package main

import (
	"errors"
	"fmt"
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

	err := s.config.SetUser(cmd.args[0])
	if err != nil {
		return err
	}

	fmt.Printf("username has been updated to %s\n", cmd.args[0])
	return nil
}
