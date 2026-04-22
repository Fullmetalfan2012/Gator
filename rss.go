package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	"github.com/Fullmetalfan2012/Gator/internal/database"
	"github.com/google/uuid"
)

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

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("User-Agent", "gator")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching feed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var feed RSSFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("error unmarshaling feed: %w", err)
	}

	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)
	for i := range feed.Channel.Item {
		feed.Channel.Item[i].Title = html.UnescapeString(feed.Channel.Item[i].Title)
		feed.Channel.Item[i].Description = html.UnescapeString(feed.Channel.Item[i].Description)
	}

	return &feed, nil
}

func handlerAgg(s *state, cmd command) error {
	feed, err := fetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return fmt.Errorf("error fetching feed: %w", err)
	}
	fmt.Printf("%+v\n", feed)
	return nil
}

func createFeedFollow(s *state, userID, feedID uuid.UUID) (database.CreateFeedFollowRow, error) {
	now := time.Now()
	rows, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    userID,
		FeedID:    feedID,
	})
	if err != nil {
		return database.CreateFeedFollowRow{}, fmt.Errorf("error creating feed follow: %w", err)
	}
	if len(rows) == 0 {
		return database.CreateFeedFollowRow{}, fmt.Errorf("no feed follow record returned")
	}
	return rows[0], nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("addfeed requires two arguments: name url")
	}
	name := cmd.args[0]
	url := cmd.args[1]

	now := time.Now()
	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})
	if err != nil {
		return fmt.Errorf("error creating feed: %w", err)
	}

	fmt.Printf("Feed created successfully!\n")
	fmt.Printf("ID:        %s\n", feed.ID)
	fmt.Printf("CreatedAt: %s\n", feed.CreatedAt)
	fmt.Printf("UpdatedAt: %s\n", feed.UpdatedAt)
	fmt.Printf("Name:      %s\n", feed.Name)
	fmt.Printf("URL:       %s\n", feed.Url)
	fmt.Printf("UserID:    %s\n", feed.UserID)

	follow, err := createFeedFollow(s, user.ID, feed.ID)
	if err != nil {
		return err
	}
	fmt.Printf("Now following: %s (user: %s)\n", follow.FeedName, follow.UserName)
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("follow requires one argument: url")
	}
	url := cmd.args[0]

	feed, err := s.db.GetFeedByURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("error finding feed with URL %s: %w", url, err)
	}

	follow, err := createFeedFollow(s, user.ID, feed.ID)
	if err != nil {
		return err
	}
	fmt.Printf("Feed:  %s\nUser:  %s\n", follow.FeedName, follow.UserName)
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("error fetching feed follows: %w", err)
	}
	for _, f := range follows {
		fmt.Printf("* %s\n", f.FeedName)
	}
	return nil
}


func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("unfollow requires one argument: url")
	}
	err := s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		Url:    cmd.args[0],
		UserID: user.ID,
	})
	if err != nil {
		return fmt.Errorf("error unfollowing feed: %w", err)
	}
	fmt.Printf("Unfollowed %s\n", cmd.args[0])
	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("error fetching feeds: %w", err)
	}
	for _, f := range feeds {
		fmt.Printf("Name: %s\nURL:  %s\nUser: %s\n\n", f.Name, f.Url, f.UserName)
	}
	return nil
}