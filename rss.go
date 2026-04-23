package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Fullmetalfan2012/Gator/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
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

func scrapeFeeds(s *state) {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		fmt.Printf("error getting next feed: %v\n", err)
		return
	}

	err = s.db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		fmt.Printf("error marking feed fetched: %v\n", err)
		return
	}

	rssFeed, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		fmt.Printf("error fetching feed %s: %v\n", feed.Url, err)
		return
	}

	fmt.Printf("Feed: %s\n", feed.Name)
	for _, item := range rssFeed.Channel.Item {
		publishedAt := sql.NullTime{}
		if t, err := parsePubDate(item.PubDate); err == nil {
			publishedAt = sql.NullTime{Time: t, Valid: true}
		}

		description := sql.NullString{}
		if item.Description != "" {
			description = sql.NullString{String: item.Description, Valid: true}
		}

		now := time.Now()
		_, err := s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   now,
			UpdatedAt:   now,
			Title:       item.Title,
			Url:         item.Link,
			Description: description,
			PublishedAt: publishedAt,
			FeedID:      feed.ID,
		})
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				continue
			}
			log.Printf("error saving post %q: %v", item.Link, err)
		}
	}
}

func parsePubDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty pubDate")
	}
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC3339,
		time.RFC3339Nano,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized pubDate format: %q", s)
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("agg requires one argument: time_between_reqs")
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", cmd.args[0], err)
	}

	fmt.Printf("Collecting feeds every %v\n", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
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

func handlerBrowse(s *state, cmd command, user database.User) error {
	limit := int32(2)
	if len(cmd.args) >= 1 {
		n, err := strconv.ParseInt(cmd.args[0], 10, 32)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid limit %q: must be a positive integer", cmd.args[0])
		}
		limit = int32(n)
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		return fmt.Errorf("error fetching posts: %w", err)
	}

	if len(posts) == 0 {
		fmt.Println("No posts found. Try running `agg` to fetch some!")
		return nil
	}

	for _, p := range posts {
		published := "(unknown)"
		if p.PublishedAt.Valid {
			published = p.PublishedAt.Time.Format(time.RFC1123)
		}
		fmt.Printf("=== %s ===\n", p.Title)
		fmt.Printf("Published: %s\n", published)
		fmt.Printf("URL:       %s\n", p.Url)
		if p.Description.Valid && p.Description.String != "" {
			fmt.Printf("%s\n", p.Description.String)
		}
		fmt.Println()
	}
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