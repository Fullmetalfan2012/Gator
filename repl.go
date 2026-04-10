package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Fullmetalfan2012/Gator/internal/config"
	"github.com/Fullmetalfan2012/Gator/internal/database"
	"github.com/google/uuid"
)

type state struct {
	db  *database.Queries
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
	_, err := s.db.GetUser(context.Background(), cmd.args[0])
	if err != nil {
		fmt.Println("Error: user does not exist")
		os.Exit(1)
	}
	err = s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return fmt.Errorf("Error setting user in login handler: %w", err)
	}
	fmt.Println("User has been set.")
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if cmd.args == nil || len(cmd.args) != 1 {
		return fmt.Errorf("Error: Register expects one argument: name")
	}
	name := cmd.args[0]
	now := time.Now()
	usr, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name:      name,
	})
	if err != nil {
		fmt.Printf("Error: user with name '%s' already exists\n", name)
		os.Exit(1)
	}
	err = s.cfg.SetUser(name)
	if err != nil {
		return fmt.Errorf("Error setting current user: %w", err)
	}
	fmt.Printf("User created successfully!\n")
	fmt.Printf("User data: %+v\n", usr)
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

func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteUsers(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resetting database: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Database reset successfully.")
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.cmds[name] = f
	return
}