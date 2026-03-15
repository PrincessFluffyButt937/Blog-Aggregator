package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/PrincessFluffyButt937/Blog-Aggregator/internal/config"
	"github.com/PrincessFluffyButt937/Blog-Aggregator/internal/database"
	"github.com/google/uuid"
)

type state struct {
	con *config.Config
	db  *database.Queries
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
		fmt.Println("handlerLogin error: No args")
		os.Exit(1)
	}
	_, err := s.db.GetUser(context.Background(), sql.NullString{String: c.arg[0], Valid: true})
	if err == sql.ErrNoRows {
		fmt.Printf("handlerLogin error: User %s does not exist!\n", c.arg[0])
		os.Exit(1)
	}
	s.con.Current_user_name = c.arg[0]
	fmt.Printf("User %s has been set.\n", c.arg[0])
	return nil
}

func handlerRegister(s *state, c command) error {
	if len(c.arg) == 0 {
		fmt.Println("handlerLogin error: No args")
		os.Exit(1)
	}
	user_data := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      sql.NullString{String: c.arg[0], Valid: true},
	}
	_, err := s.db.GetUser(context.Background(), sql.NullString{String: c.arg[0], Valid: true})
	if err != sql.ErrNoRows {
		fmt.Printf("handlerRegister error: User %s already exists!\n", c.arg[0])
		os.Exit(1)
	}
	s.db.CreateUser(context.Background(), user_data)
	s.con.Current_user_name = c.arg[0]
	fmt.Printf("User %s was successfuly registered.\n", c.arg[0])
	return nil
}

func handlerReset(s *state, c command) error {
	err := s.db.DelUsers(context.Background())
	if err != nil {
		fmt.Printf("handlerReset error: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Users table reset was successul.")
	return nil
}
