package main

import (
	"fmt"
	"os"

	"github.com/PrincessFluffyButt937/Blog-Aggregator/internal/config"
)

func main() {
	args := os.Args
	Cfg_path, _ := config.Get_cfg_path()
	Cfg, _ := config.Read_cfg(Cfg_path)
	s := state{
		con: Cfg,
	}
	coms := commands{
		repo: make(map[string]func(*state, command) error),
	}
	coms.register("login", handlerLogin)
	if len(args) < 3 {
		fmt.Println("Error, # of Args is less than 2")
		os.Exit(1)
		return
	}
	com := command{
		name: args[1],
		arg:  args[2:],
	}
	coms.run(&s, com)
	if err := config.Write_cfg(s.con); err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}
	os.Exit(0)

}
