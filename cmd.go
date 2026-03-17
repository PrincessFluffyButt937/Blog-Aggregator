package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
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

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
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

func handlerUsers(s *state, c command) error {
	user_entries, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Printf("handlerUsers error: %s\n", err)
		os.Exit(1)
	}
	current_user := s.con.Current_user_name
	for _, user := range user_entries {
		db_user := user.Name.String
		if db_user == current_user {
			fmt.Printf("* %s (current)\n", db_user)
		} else {
			fmt.Printf("* %s\n", db_user)
		}
	}
	return nil
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		fmt.Printf("fetchFeed error - NewRequest..: %s\n", err)
		os.Exit(1)
	}
	req.Header.Set("User-Agent", "gator")

	cli := http.Client{}

	res, err := cli.Do(req)
	if err != nil {
		fmt.Printf("fetchFeed error - request: %s\n", err)
		os.Exit(1)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("fetchFeed error - io.ReadAll: %s\n", err)
		os.Exit(1)
	}
	var feed *RSSFeed

	if err := xml.Unmarshal(body, &feed); err != nil {
		fmt.Printf("fetchFeed error - unmarshal: %s\n", err)
		os.Exit(1)
	}
	return feed, nil
}
