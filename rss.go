package main

import (
	"context"
	"encoding/xml"
	"errors"
	"html"
	"io"
	"net/http"
	"time"

	"github.com/carsondecker/gator/internal/database"
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
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gator")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}

	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)
	for _, item := range feed.Channel.Item {
		item.Title = html.UnescapeString(item.Title)
		item.Description = html.UnescapeString(item.Description)
	}

	return &feed, nil
}

func scrapeFeeds(s *state, time_between_reqs time.Duration) error {
	ticker := time.NewTicker(time_between_reqs)

	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}
	if len(feeds) == 0 {
		return errors.New("no feeds to aggregate")
	}

	for {
		nextFeed, err := s.db.GetNextFeedToFetch(context.Background())
		if err != nil {
			return err
		}

		err = s.db.MarkFeedFetched(context.Background(), nextFeed.ID)
		if err != nil {
			return err
		}

		feed, err := fetchFeed(context.Background(), nextFeed.Url)
		if err != nil {
			return err
		}

		for _, item := range feed.Channel.Item {
			layouts := []string{
				"Mon, 02 Jan 2006 15:04:05 -0700",
				"Mon, 02 Jan 2006 15:04:05 -0700 (MST)",
				"Mon, 02 Jan 2006 15:04:05 MST",
				"Mon, 02 Jan 2006 15:04:05 Z0700",
				time.RFC1123Z,
				time.RFC822Z,
			}

			var parsedTime time.Time
			var parseErr error
			success := false

			for _, layout := range layouts {
				parsedTime, parseErr = time.Parse(layout, item.PubDate)
				if parseErr == nil {
					success = true
					break
				}
			}
			if !success {
				parsedTime = time.Time{}
			}

			if err != nil {
				return err
			}

			_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
				ID:          uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Title:       item.Title,
				Url:         item.Link,
				Description: item.Description,
				PublishedAt: parsedTime,
				FeedID:      nextFeed.ID,
			})

			if err != nil {
				if pqErr, ok := err.(*pq.Error); ok {
					if pqErr.Code == "23505" && pqErr.Constraint == "posts_url_key" {
						break
					}
				}
				return err
			}
		}

		<-ticker.C
	}
}
