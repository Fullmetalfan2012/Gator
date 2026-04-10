package main

import (
	"fmt"
	"github.com/Fullmetalfan2012/Gator/internal/config"
)

type state struct {
	cfg *config.Config
}

type command struct {
	name	string
	args	[]string
}

func handlerLogin(s *state, cmd command) error {
	if cmd.args == nil || len(cmd.args) != 1 {
		return fmt.Errorf("Error: Login expects one argument: username")
	}
	err := s.cfg.SetUser(cmd.args[0]) 
	if err != nil {
		return fmt.Errorf("Error setting user in login handler: %w", err)
	}
	fmt.Println("User has been set.")
	return nil
}

type commands struct {
	cmds map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	toRun, ok := c.cmds[cmd.name]
	if !ok {
		return fmt.Errorf("Error: Command not found!")
	}
	err := toRun(s, cmd)
	if err != nil {
		return fmt.Errorf("Error running command %v: %w", cmd.name, err)
	}
	return nil	
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.cmds[name] = f
	return
}