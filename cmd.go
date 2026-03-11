package main

import (
	"fmt"

	"github.com/PrincessFluffyButt937/Blog-Aggregator/internal/config"
)

type state struct {
	con *config.Config
}

type command struct {
	name string
	arg  []string
}

type commands struct {
	repo map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	f_call, exists := c.repo[cmd.name]
	if !exists {
		return fmt.Errorf("commands run: command %s does not exist", cmd.name)
	}
	err := f_call(s, cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.repo[name] = f
}

func handlerLogin(s *state, c command) error {
	if len(c.arg) == 0 {
		return fmt.Errorf("handlerLogin error: No args")
	}
	s.con.Current_user_name = c.arg[0]
	fmt.Printf("User %s has been set.\n", c.arg[0])
	return nil
}
