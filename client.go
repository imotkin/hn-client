package hn

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	version = "0.0.1"

	baseURL   = "https://hacker-news.firebaseio.com/v0"
	userAgent = "hn-client" + "/" + version

	StoryType      = "story"
	CommentType    = "comment"
	AskType        = "ask"
	JobType        = "job"
	PollType       = "poll"
	PollOptionType = "pollopt"
)

var (
	ErrNotFound = errors.New("item is not found")

	maxWorkers = -1

	defaultClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:    100,
			MaxConnsPerHost: 100,
			IdleConnTimeout: 90 * time.Second,
		},
	}
)

// SetMaxWorkers sets the maximum number of workers for multiple item fetch operations.
// The default value of maxWorkers is -1, meaning there is no limit to the number of workers.
func SetMaxWorkers(n int) {
	maxWorkers = n
}

// Client represents a client for the Hacker News API.
type Client struct {
	Items *ItemService
	Users *UserService
	Live  *LiveService
}

// NewClient returns a new Hacker News API client. If httpClient is nil, the default client will be used.
func NewClient(httpClient *http.Client) *Client {
	httpClient = cmp.Or(httpClient, defaultClient)

	var (
		items = &ItemService{client: httpClient}
		users = &UserService{client: httpClient, items: items}
		live  = &LiveService{client: httpClient, items: items}
	)

	return &Client{
		Items: items,
		Users: users,
		Live:  live,
	}
}

// baseItem is a base type for all items, containing only the fields common to all items.
type baseItem struct {
	ID    uint      `json:"id,omitempty"`
	By    string    `json:"by,omitempty"`
	Score int       `json:"score,omitempty"`
	Time  Timestamp `json:"time,omitzero"`
	Type  string    `json:"type,omitempty"`
}

func (i baseItem) getID() uint {
	return i.ID
}

func (i baseItem) getBy() string {
	return i.By
}

func (i baseItem) getScore() int {
	return i.Score
}

func (i baseItem) getTime() Timestamp {
	return i.Time
}

func (i baseItem) getType() string {
	return i.Type
}

// Item is a common type for all other types: story, comment, poll, etc.
// It contains all fields, so some of them may be empty if the value of the
// specific type doesn't have that field.
type Item struct {
	baseItem

	Descendants int    `json:"descendants,omitempty"`
	Parts       []uint `json:"parts,omitempty"`
	Parent      uint   `json:"parent,omitempty"`
	Kids        []uint `json:"kids,omitempty"`
	Text        string `json:"text,omitempty"`
	Title       string `json:"title,omitempty"`
	Poll        uint   `json:"poll,omitempty"`
	URL         string `json:"url,omitempty"`
}

type Story struct {
	baseItem

	Descendants int    `json:"descendants,omitempty"`
	Kids        []uint `json:"kids,omitempty"`
	Text        string `json:"text,omitempty"`
	Title       string `json:"title,omitempty"`
	URL         string `json:"url,omitempty"`
}

func (s Story) Type() string {
	return StoryType
}

type Comment struct {
	baseItem

	Kids   []uint `json:"kids,omitempty"`
	Parent uint   `json:"parent,omitempty"`
	Text   string `json:"text,omitempty"`
}

func (c Comment) Type() string {
	return CommentType
}

type Ask struct {
	baseItem

	Descendants int    `json:"descendants,omitempty"`
	Kids        []uint `json:"kids,omitempty"`
	Text        string `json:"text,omitempty"`
	Title       string `json:"title,omitempty"`
}

func (a Ask) Type() string {
	return AskType
}

type Job struct {
	baseItem

	Text  string `json:"text,omitempty"`
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}

func (j Job) Type() string {
	return JobType
}

type Poll struct {
	baseItem

	Descendants int    `json:"descendants,omitempty"`
	Kids        []uint `json:"kids,omitempty"`
	Parts       []uint `json:"parts,omitempty"`
	Text        string `json:"text,omitempty"`
	Title       string `json:"title,omitempty"`
}

func (p Poll) Type() string {
	return PollType
}

type PollOption struct {
	baseItem

	Poll uint   `json:"poll,omitempty"`
	Text string `json:"text,omitempty"`
}

func (o PollOption) Type() string {
	return PollOptionType
}

type User struct {
	ID        string    `json:"id,omitempty"`
	About     string    `json:"about,omitempty"`
	Created   Timestamp `json:"created,omitzero"`
	Karma     int       `json:"karma,omitempty"`
	Submitted []uint    `json:"submitted,omitempty"`
}

type Update struct {
	Items    []uint   `json:"items,omitempty"`
	Profiles []string `json:"profiles,omitempty"`
}

type Timestamp struct {
	time.Time
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var timestamp int64

	err := json.Unmarshal(data, &timestamp)
	if err != nil {
		return err
	}

	t.Time = time.Unix(timestamp, 0)

	return nil
}

// ItemService provides methods to retrieve data about Hacker News items.
type ItemService struct {
	client *http.Client
}

// Get return an Item with the specified ID.
func (s *ItemService) Get(ctx context.Context, id uint) (Item, error) {
	item, err := Fetch[Item](ctx, s.client, http.MethodGet, fmt.Sprintf("/item/%d", id))
	if err != nil {
		return Item{}, err
	}

	if item.Text != "" {
		item.Text = html.UnescapeString(item.Text)
	}

	if item.Title != "" {
		item.Title = html.UnescapeString(item.Title)
	}

	return item, nil
}

// List returns a list of items with specific IDs, filtered if necessary.
func (s *ItemService) List(ctx context.Context, ids []uint, filter func(Item) bool) ([]Item, error) {
	if len(ids) == 0 {
		return []Item{}, nil
	}

	var (
		processed = make(chan Item, 10)
		wait      = make(chan struct{})
		items     = make([]Item, 0, len(ids))
		itemsMap  = make(map[uint]Item, len(ids))
	)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxWorkers)

	go func() {
		defer close(wait)

		for {
			select {
			case item, ok := <-processed:
				if ok {
					itemsMap[item.ID] = item
				} else {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for _, id := range ids {
		g.Go(func() error {
			item, err := s.Get(ctx, id)
			if err != nil {
				return err
			}

			if filter != nil && filter(item) {
				processed <- item
			} else if filter == nil {
				processed <- item
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	close(processed)

	<-wait

	for _, id := range ids {
		if item, ok := itemsMap[id]; ok {
			items = append(items, item)
		}
	}

	return items, nil
}

// UserService provides methods to retrieve data about Hacker News users.
type UserService struct {
	client *http.Client
	items  *ItemService
}

// Get returns a User with the given name.
func (s *UserService) Get(ctx context.Context, username string) (User, error) {
	return Fetch[User](ctx, s.client, http.MethodGet, ("/user/" + username))
}

// Items returns the items submitted by the user with the given name, filtered if necessary.
func (s *UserService) Items(ctx context.Context, username string, filter func(Item) bool) ([]Item, error) {
	user, err := s.Get(ctx, username)
	if err != nil {
		return nil, err
	}

	return s.items.List(ctx, user.Submitted, filter)
}

// Comments returns the comments submitted by the user with the given name.
func (s *UserService) Comments(ctx context.Context, username string) ([]Comment, error) {
	filter := func(item Item) bool {
		return item.Type == CommentType
	}

	items, err := s.Items(ctx, username, filter)
	if err != nil {
		return nil, err
	}

	return ToList[Comment](items), nil
}

// Stories returns the stories submitted by the user with the given name.
func (s *UserService) Stories(ctx context.Context, username string) ([]Story, error) {
	filter := func(item Item) bool {
		return item.Type == StoryType
	}

	items, err := s.Items(ctx, username, filter)

	if err != nil {
		return nil, err
	}

	return ToList[Story](items), nil
}

// Jobs returns the jobs submitted by the user with the given name.
func (s *UserService) Jobs(ctx context.Context, username string) ([]Job, error) {
	items, err := s.Items(ctx, username, func(item Item) bool {
		return item.Type == JobType
	})

	if err != nil {
		return nil, err
	}

	return ToList[Job](items), nil
}

// Asks returns the asks submitted by the user with the given name.
func (s *UserService) Asks(ctx context.Context, username string) ([]Ask, error) {
	items, err := s.Items(ctx, username, func(item Item) bool {
		return item.Type == AskType
	})

	if err != nil {
		return nil, err
	}

	return ToList[Ask](items), nil
}

// Polls returns the polls submitted by the user with the given name.
func (s *UserService) Polls(ctx context.Context, username string) ([]Poll, error) {
	items, err := s.Items(ctx, username, func(item Item) bool {
		return item.Type == PollType
	})

	if err != nil {
		return nil, err
	}

	return ToList[Poll](items), nil
}

// PollOptions returns the poll options submitted by the user with the given name.
func (s *UserService) PollOptions(ctx context.Context, username string) ([]PollOption, error) {
	items, err := s.Items(ctx, username, func(item Item) bool {
		return item.Type == PollOptionType
	})

	if err != nil {
		return nil, err
	}

	return ToList[PollOption](items), nil
}

// LiveService provides methods to retrieve data about recent updates.
type LiveService struct {
	client *http.Client
	items  *ItemService
}

// Recent returns the latest items with the given offset.
func (s *LiveService) Recent(ctx context.Context, offset uint) ([]Item, error) {
	latest, err := s.MaxID(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]uint, 0, offset)
	for i := latest - offset; i <= latest; i++ {
		ids = append(ids, i)
	}

	return s.items.List(ctx, ids, nil)
}

// MaxID returns the ID of the most recently published item.
func (s *LiveService) MaxID(ctx context.Context) (uint, error) {
	return Fetch[uint](ctx, s.client, http.MethodGet, "/maxitem")
}

// New returns a list of IDs for the new stories.
func (s *LiveService) New(ctx context.Context) ([]uint, error) {
	return Fetch[[]uint](ctx, s.client, http.MethodGet, "/newstories")
}

// NewList returns a list of items for the new stories, filtered if necessary.
func (s *LiveService) NewList(ctx context.Context, filter func(Item) bool) ([]Item, error) {
	ids, err := s.New(ctx)
	if err != nil {
		return nil, err
	}

	return s.items.List(ctx, ids, filter)
}

// Top returns a list of IDs for the top stories.
func (s *LiveService) Top(ctx context.Context) ([]uint, error) {
	return Fetch[[]uint](ctx, s.client, http.MethodGet, "/topstories")
}

// TopList returns a list of items for the top stories, filtered if necessary.
func (s *LiveService) TopList(ctx context.Context, filter func(Item) bool) ([]Item, error) {
	ids, err := s.Top(ctx)
	if err != nil {
		return nil, err
	}

	return s.items.List(ctx, ids, filter)
}

// Best returns a list of IDs for the best stories.
func (s *LiveService) Best(ctx context.Context) ([]uint, error) {
	return Fetch[[]uint](ctx, s.client, http.MethodGet, "/beststories")
}

// BestList returns a list of items for the best stories, filtered if necessary.
func (s *LiveService) BestList(ctx context.Context, filter func(Item) bool) ([]Item, error) {
	ids, err := s.Best(ctx)
	if err != nil {
		return nil, err
	}

	return s.items.List(ctx, ids, filter)
}

// Ask returns a list of IDs for the asks.
func (s *LiveService) Ask(ctx context.Context) ([]uint, error) {
	return Fetch[[]uint](ctx, s.client, http.MethodGet, "/askstories")
}

// AskList returns a list of items for the asks, filtered if necessary.
func (s *LiveService) AskList(ctx context.Context, filter func(Item) bool) ([]Ask, error) {
	ids, err := s.Ask(ctx)
	if err != nil {
		return nil, err
	}

	items, err := s.items.List(ctx, ids, filter)
	if err != nil {
		return nil, err
	}

	return ToList[Ask](items), nil
}

// Show returns a list of IDs for the shows.
func (s *LiveService) Show(ctx context.Context) ([]uint, error) {
	return Fetch[[]uint](ctx, s.client, http.MethodGet, "/showstories")
}

// ShowList returns a list of items for the shows, filtered if necessary.
func (s *LiveService) ShowList(ctx context.Context, filter func(Item) bool) ([]Story, error) {
	ids, err := s.Show(ctx)
	if err != nil {
		return nil, err
	}

	items, err := s.items.List(ctx, ids, filter)
	if err != nil {
		return nil, err
	}

	return ToList[Story](items), nil
}

// Job returns a list of IDs for the jobs.
func (s *LiveService) Job(ctx context.Context) ([]uint, error) {
	return Fetch[[]uint](ctx, s.client, http.MethodGet, "/jobstories")
}

// JobList returns a list of items for the jobs, filtered if necessary.
func (s *LiveService) JobList(ctx context.Context, filter func(Item) bool) ([]Job, error) {
	ids, err := s.Job(ctx)
	if err != nil {
		return nil, err
	}

	items, err := s.items.List(ctx, ids, filter)
	if err != nil {
		return nil, err
	}

	return ToList[Job](items), nil
}

// Update returns an Update containing IDs of updated items and profiles.
func (s *LiveService) Update(ctx context.Context) (Update, error) {
	return Fetch[Update](ctx, s.client, http.MethodGet, "/updates")
}

// UpdateList returns a list of updated items, filtered if necessary.
func (s *LiveService) UpdateList(ctx context.Context, filter func(Item) bool) ([]Item, error) {
	update, err := s.Update(ctx)
	if err != nil {
		return nil, err
	}

	return s.items.List(ctx, update.Items, filter)
}

// Fetch sends an HTTP request to the Hacker News API and returns a value of the specified type.
func Fetch[T any](ctx context.Context, client *http.Client, method, url string) (T, error) {
	var t T

	req, err := http.NewRequestWithContext(ctx, method, (baseURL + url + ".json"), nil)
	if err != nil {
		return t, fmt.Errorf("create HTTP request: %w", err)
	}

	req.Header.Add("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return t, fmt.Errorf("send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return t, fmt.Errorf("read response JSON: %w", err)
	}

	if string(body) == "null" {
		return t, ErrNotFound
	}

	err = json.Unmarshal(body, &t)
	if err != nil {
		return t, fmt.Errorf("decode response JSON: %w", err)
	}

	return t, nil
}
