package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
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
		return fmt.Errorf("handlerLogin error: No args")
	}
	_, err := s.db.GetUser(context.Background(), sql.NullString{String: c.arg[0], Valid: true})
	if err == sql.ErrNoRows {
		return fmt.Errorf("handlerLogin error: User %s does not exist!\n", c.arg[0])
	}
	s.con.Current_user_name = c.arg[0]
	fmt.Printf("User %s has been set.\n", c.arg[0])
	return nil
}

func handlerRegister(s *state, c command) error {
	if len(c.arg) == 0 {
		return fmt.Errorf("handlerLogin error: No args")
	}
	user_data := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      sql.NullString{String: c.arg[0], Valid: true},
	}
	_, err := s.db.GetUser(context.Background(), sql.NullString{String: c.arg[0], Valid: true})
	if err != sql.ErrNoRows {
		return fmt.Errorf("handlerRegister error: User %s already exists!\n", c.arg[0])
	}
	s.db.CreateUser(context.Background(), user_data)
	s.con.Current_user_name = c.arg[0]
	fmt.Printf("User %s was successfuly registered.\n", c.arg[0])
	return nil
}

func handlerReset(s *state, c command) error {
	err := s.db.DelUsers(context.Background())
	if err != nil {
		return fmt.Errorf("handlerReset error: %s\n", err)
	}
	fmt.Println("Users table reset was successul.")
	return nil
}

func handlerUsers(s *state, c command) error {
	user_entries, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("handlerUsers error: %s\n", err)
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
		return nil, fmt.Errorf("fetchFeed error - NewRequest..: %s\n", err)
	}
	req.Header.Set("User-Agent", "gator")
	cli := http.Client{}

	res, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetchFeed error - request: %s\n", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("fetchFeed error - io.ReadAll: %s\n", err)
	}

	var feed RSSFeed

	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("fetchFeed error - unmarshal: %s\n", err)
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
	if len(c.arg) < 1 {
		return fmt.Errorf("handlerAgg error: 1 arg is required\nplease enter agr: <time>\nValid formats:\n 30s; 5m; 1h...\n")
	}
	cycle_duration, err := time.ParseDuration(c.arg[0])
	if err != nil {
		return fmt.Errorf("handlerAgg error - invalid time format: %s\n", err)
	}
	fmt.Printf("Collecting feeds every %s\n", c.arg[0])
	ticker := time.NewTicker(cycle_duration)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func handlerAddfeed(s *state, c command, user database.User) error {
	if len(c.arg) < 2 {
		return fmt.Errorf("handlerAddfeed error: 2 args are required\nplease enter agrs in order: <feed_name> <feed_url>\n")
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
		return fmt.Errorf("handlerAddfeed error db_CreateFeed: %s\n", err)
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
		return fmt.Errorf("handlerFeeds error db_GetFeeds: %s\n", err)
	}
	for _, row := range rows {
		user, err := s.db.GetUserById(context.Background(), row.UserID)
		if err != nil {
			return fmt.Errorf("handlerFeeds error db_GetUserById: %s\n", err)
		}
		fmt.Printf("|user_name: %s |feed_ID: %v | F_name: %s | F_url: %s| F_created: %v | F_updated: %v |\n",
			user.Name.String, row.ID, row.Name, row.Url, row.CreatedAt, row.UpdatedAt)
	}
	return nil
}

func handlerFollow(s *state, c command, user database.User) error {
	if len(c.arg) < 1 {
		return fmt.Errorf("handlerFollow error: 1 arg is required\nplease enter agr: <url>\n")
	}
	feed_row, err := s.db.GetFeedByUrl(context.Background(), c.arg[0])
	if err != nil {
		return fmt.Errorf("handlerFollow error db_GetFeedByUrl: %s\n", err)
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

func handlerFollowing(s *state, c command, user database.User) error {
	follows, err := s.db.GetFollows(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("handlerFollowing error db_GetFollows: %s\n", err)
	}
	fmt.Println("You are following these feeds:")
	for _, follow := range follows {
		fmt.Printf("* %s\n", follow.FeedName)
	}
	return nil
}

func handlerUnfollow(s *state, c command, user database.User) error {
	if len(c.arg) < 1 {
		return fmt.Errorf("handlerFollow error: 1 arg is required\nplease enter agr: <url>\n")
	}
	feed, err := s.db.GetFeedByUrl(context.Background(), c.arg[0])
	if err != nil {
		return fmt.Errorf("handlerUnfollow error db_GetFeedByUrl: %s\n", err)
	}

	follow := database.DelFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}

	if err := s.db.DelFollow(context.Background(), follow); err != nil {
		return fmt.Errorf("handlerUnfollow error db_DelFollow: %s\n", err)
	}
	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), sql.NullString{String: s.con.Current_user_name, Valid: true})
		if err != nil {
			return fmt.Errorf("middlewareLoggedIn error db_GetUser: %s\n", err)
		}
		return handler(s, cmd, user)
	}
}

func scrapeFeeds(s *state) {
	feed_to_update, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		fmt.Printf("scrapeFeeds error db_GetNextFeedToFetch: %s\n", err)
	}
	now := time.Now()
	feed_to_mark := database.MarkFeedFetchedParams{
		UpdatedAt:     now,
		LastFetchedAt: sql.NullTime{Time: now, Valid: true},
		ID:            feed_to_update.ID,
	}
	if err := s.db.MarkFeedFetched(context.Background(), feed_to_mark); err != nil {
		fmt.Printf("scrapeFeeds error db_MarkFeedFetched: %s\n", err)
	}
	feed, err := fetchFeed(context.Background(), feed_to_update.Url)
	if err != nil {
		fmt.Printf("scrapeFeeds error fetchFeed: %s\n", err)
	} else {
		for _, item := range feed.Channel.Item {
			post_param := database.CreatePostParams{
				ID:          uuid.New(),
				CreatedAt:   now,
				UpdatedAt:   now,
				Title:       item.Title,
				Url:         item.Link,
				Description: item.Description,
				FeedID:      feed_to_update.ID,
			}
			fmt.Println("-----------------")
			fmt.Println(item.Description)
			published, err := time.Parse(time.RFC1123Z, item.PubDate)
			if err != nil {
				fmt.Printf("unknown time format: %s error: %s\n", item.PubDate, err)
				post_param.PublishedAt = sql.NullTime{Valid: false}
			} else {
				post_param.PublishedAt = sql.NullTime{Time: published, Valid: true}
			}
			if _, err := s.db.CreatePost(context.Background(), post_param); err != nil {
				if !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
					fmt.Printf("scrapeFeeds error db_CreatePost: %s\n", err)
				}
			}
		}
	}
}

func handlerBrowse(s *state, c command, user database.User) error {
	limit := 2
	if len(c.arg) >= 1 {
		i, err := strconv.Atoi(c.arg[0])
		if err == nil {
			limit = i
		}
	}
	params := database.GetPostsParams{
		UserID: user.ID,
		Limit:  int32(limit),
	}
	posts, err := s.db.GetPosts(context.Background(), params)
	if err != nil {
		return fmt.Errorf("handlerBrowse error db_GetPosts: %s\n", err)
	}
	fmt.Println("Printing your posts...")
	for _, post := range posts {
		fmt.Println("--------------------------------------------------------------")
		fmt.Printf("Post title: %s\n", post.Title)
		fmt.Println(post.Description)
	}
	return nil
}
