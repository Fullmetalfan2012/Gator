package main

import (
	"log"
	"os"
	"github.com/Fullmetalfan2012/Gator/internal/config"
)

func main() {
	s := state{}
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}
	s.cfg = &cfg
	cmds := commands{cmds: make(map[string]func(*state, command) error)}
	cmds.register("login", handlerLogin)
	input := os.Args
	if len(input) < 2 {
		log.Fatalf("Too few arguments")
	}
	cmd := command{name: input[1], args: input[2:]}
	err = cmds.run(&s, cmd)
	if err != nil {
		log.Fatalf("Error running command %v: %v", cmd.name, err)
	}
}