package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
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
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("fetchFeed error - io.ReadAll: %s\n", err)
		os.Exit(1)
	}

	var feed RSSFeed

	if err := xml.Unmarshal(body, &feed); err != nil {
		fmt.Printf("fetchFeed error - unmarshal: %s\n", err)
		os.Exit(1)
	}
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)
	for i, item := range feed.Channel.Item {
		feed.Channel.Item[i].Title = html.UnescapeString(item.Title)
		feed.Channel.Item[i].Description = html.UnescapeString(item.Description)
	}

	return &feed, nil
}

func handlerAgg(s *state, c command) error {
	agg, err := fetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		fmt.Printf("handlerAgg error: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(agg)
	return nil
}

func handlerAddfeed(s *state, c command) error {
	user, err := s.db.GetUser(context.Background(), sql.NullString{String: s.con.Current_user_name, Valid: true})
	if err == sql.ErrNoRows {
		fmt.Printf("handlerAddfeed error: User %s does not exist!\n", s.con.Current_user_name)
		os.Exit(1)
	}
	if len(c.arg) < 2 {
		fmt.Println("handlerAddfeed error: 2 args are required")
		fmt.Println("please enter agrs in order: <feed_name> <feed_url>")
		os.Exit(1)
	}
	now := time.Now()

	feed_data := database.CreateFeedParams{
		ID:        uuid.New(),
		Name:      c.arg[0],
		Url:       c.arg[1],
		UserID:    user.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	feed, err := s.db.CreateFeed(context.Background(), feed_data)
	if err != nil {
		fmt.Printf("handlerAddfeed error db_CreateFeed: %s\n", err)
		os.Exit(1)
	}
	follow_data := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: feed.CreatedAt,
		UpdatedAt: feed.UpdatedAt,
		UserID:    user.ID,
		FeedID:    feed.ID,
	}
	s.db.CreateFeedFollow(context.Background(), follow_data)
	return nil
}

func handlerFeeds(s *state, c command) error {
	rows, err := s.db.GetFeeds(context.Background())
	if err != nil {
		fmt.Printf("handlerFeeds error db_GetFeeds: %s\n", err)
		os.Exit(1)
	}
	for _, row := range rows {
		user, err := s.db.GetUserById(context.Background(), row.UserID)
		if err != nil {
			fmt.Printf("handlerFeeds error db_GetUserById: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("|user_name: %s |feed_ID: %v | F_name: %s | F_url: %s| F_created: %v | F_updated: %v |\n",
			user.Name.String, row.ID, row.Name, row.Url, row.CreatedAt, row.UpdatedAt)
	}
	return nil
}

func handlerFollow(s *state, c command) error {
	if len(c.arg) < 1 {
		fmt.Println("handlerFollow error: 1 arg is required")
		fmt.Println("please enter agr: <url>")
		os.Exit(1)
	}
	feed_row, err := s.db.GetFeedByUrl(context.Background(), c.arg[0])
	if err != nil {
		fmt.Printf("handlerFollow error db_GetFeedByUrl: %s\n", err)
		os.Exit(1)
	}

	user, err := s.db.GetUser(context.Background(), sql.NullString{String: s.con.Current_user_name, Valid: true})
	if err != nil {
		fmt.Printf("handlerFollow error db_GetUser: %s\n", err)
		os.Exit(1)
	}

	now := time.Now()

	feed := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    user.ID,
		FeedID:    feed_row.ID,
	}
	s.db.CreateFeedFollow(context.Background(), feed)
	fmt.Printf("User %s is now following %s feed.\n", s.con.Current_user_name, feed_row.Name)
	return nil
}

func handlerFollowing(s *state, c command) error {
	user, err := s.db.GetUser(context.Background(), sql.NullString{String: s.con.Current_user_name, Valid: true})
	if err != nil {
		fmt.Printf("handlerFollowing error db_GetUser: %s\n", err)
		os.Exit(1)
	}
	follows, err := s.db.GetFollows(context.Background(), user.ID)
	if err != nil {
		fmt.Printf("handlerFollowing error db_GetFollows: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("You are following these feeds:")
	for _, follow := range follows {
		fmt.Printf("* %s\n", follow.FeedName)
	}
	return nil
}
