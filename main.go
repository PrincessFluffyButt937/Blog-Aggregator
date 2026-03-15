package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/PrincessFluffyButt937/Blog-Aggregator/internal/config"
	"github.com/PrincessFluffyButt937/Blog-Aggregator/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	args := os.Args
	Cfg_path, _ := config.Get_cfg_path()
	Cfg, _ := config.Read_cfg(Cfg_path)
	db, err := sql.Open("postgres", Cfg.Db_url)
	if err != nil {
		fmt.Printf("Database connection error: %s\n", err)
		os.Exit(1)
		return
	}
	dbQuerries := database.New(db)

	s := state{
		con: Cfg,
		db:  dbQuerries,
	}

	// Available commands start
	coms := commands{
		repo: make(map[string]func(*state, command) error),
	}
	coms.register("login", handlerLogin)
	coms.register("register", handlerRegister)
	coms.register("reset", handlerReset)
	if len(args) < 2 {
		fmt.Println("Error, # of Args is less than 1")
		os.Exit(1)
		return
	}
	// Available commands end

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
