package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/carsondecker/gator/internal/database"
	"github.com/google/uuid"
)

type command struct {
	name string
	args []string
}

type commands struct {
	cmds map[string]func(*state, command) error
}

func initCommands() (*commands, error) {
	cmds := &commands{
		cmds: make(map[string]func(*state, command) error),
	}

	err := cmds.register("login", handlerLogin)
	if err != nil {
		return nil, err
	}

	err = cmds.register("register", handlerRegister)
	if err != nil {
		return nil, err
	}

	err = cmds.register("reset", handlerReset)
	if err != nil {
		return nil, err
	}

	err = cmds.register("users", handlerUsers)
	if err != nil {
		return nil, err
	}

	err = cmds.register("agg", handlerAgg)
	if err != nil {
		return nil, err
	}

	err = cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	if err != nil {
		return nil, err
	}

	err = cmds.register("feeds", handlerFeeds)
	if err != nil {
		return nil, err
	}

	err = cmds.register("follow", middlewareLoggedIn(handlerFollow))
	if err != nil {
		return nil, err
	}

	err = cmds.register("following", handlerFollowing)
	if err != nil {
		return nil, err
	}

	err = cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	if err != nil {
		return nil, err
	}

	err = cmds.register("browse", middlewareLoggedIn(handlerBrowse))
	if err != nil {
		return nil, err
	}

	return cmds, nil
}

func (c *commands) register(name string, f func(*state, command) error) error {
	if _, ok := c.cmds[name]; ok {
		return errors.New("command already exists")
	}

	c.cmds[name] = f
	return nil
}

func (c *commands) run(s *state, cmd command) error {
	if _, ok := c.cmds[cmd.name]; !ok {
		return errors.New("command does not exist")
	}

	return c.cmds[cmd.name](s, cmd)
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("login command expects a username argument")
	}

	if _, err := s.db.GetUser(context.Background(), cmd.args[0]); err != nil {
		return errors.New("user doesn't exists")
	}

	err := s.config.SetUser(cmd.args[0])
	if err != nil {
		return err
	}

	fmt.Printf("username has been updated to %s\n", cmd.args[0])
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("register command expects a username argument")
	}

	if _, err := s.db.GetUser(context.Background(), cmd.args[0]); err == nil {
		return errors.New("user already exists")
	}

	_, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
	})
	if err != nil {
		return err
	}

	fmt.Printf("user %s created\n", cmd.args[0])

	err = s.config.SetUser(cmd.args[0])
	if err != nil {
		return err
	}

	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.ResetUsers(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, user := range users {
		if user.Name == s.config.CurrentUserName {
			fmt.Printf("%s (current)\n", user.Name)
		} else {
			fmt.Println(user.Name)
		}
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return errors.New("agg command requires time between requests argument")
	}

	time_between_reqs, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return errors.New("could not parse entered duration")
	}

	if time_between_reqs < time.Second*10 {
		return errors.New("agg command requires at least 10s between requests")
	}

	return scrapeFeeds(s, time_between_reqs)
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return errors.New("addfeed command requires name and url arguments")
	}

	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
		Url:       cmd.args[1],
		UserID:    user.ID,
	})

	if err != nil {
		return err
	}

	fmt.Printf("feed %s created with url %s for user %s\n", feed.Name, feed.Url, user.Name)

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})

	if err != nil {
		return err
	}

	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeedsWithUser(context.Background())
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		fmt.Printf("feed %s with url %s for user %s\n", feed.Name, feed.Url, feed.UserName)
	}

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return errors.New("follow command requires url argument")
	}

	feed, err := s.db.GetFeedByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}

	feedFollow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})

	if err != nil {
		return err
	}

	fmt.Printf("user %s followed feed %s\n", feedFollow.UserName, feedFollow.FeedName)

	return nil
}

func handlerFollowing(s *state, cmd command) error {
	feedFollows, err := s.db.GetFeedFollowsForUser(context.Background(), s.config.CurrentUserName)
	if err != nil {
		return err
	}

	for _, feedFollow := range feedFollows {
		fmt.Println(feedFollow.FeedName)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return errors.New("unfollow command requires url argument")
	}

	feed, err := s.db.GetFeedByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}

	err = s.db.UnfollowFeedForUser(context.Background(), database.UnfollowFeedForUserParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})

	if err != nil {
		return err
	}

	fmt.Printf("unfollowed feed with url %s for user %s", cmd.args[0], user.Name)
	return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	browseLimit := 2
	if len(cmd.args) != 0 {
		var err error
		browseLimit, err = strconv.Atoi(cmd.args[0])
		if err != nil {
			return err
		}
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(browseLimit),
	})
	if err != nil {
		return err
	}

	for _, post := range posts {
		fmt.Println(post.Title)
	}

	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.config.CurrentUserName)
		if err != nil {
			return err
		}

		return handler(s, cmd, user)
	}
}
