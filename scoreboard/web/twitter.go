package web

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

var (
	// ErrNoAuth is an error returned by the 'NewTwitter' function when the supplied
	// credentials are nil.
	ErrNoAuth = errors.New("twitter credentials cannot be nil")
	// ErrEmptyFilter is an error returned by the 'NewTwitter' function when the supplied
	// keyword filter list is empty.
	ErrEmptyFilter = errors.New("twitter stream filter cannot be empty or nil")
)

// Tweet is a simple struct to abstract out non-important Tweet data.
type Tweet struct {
	ID        int64
	User      string
	Text      string
	Time      int64
	Images    []string
	UserName  string
	UserPhoto string
}

// Filter is a struct that allows for filtering Tweets via Test
// or Sender.
type Filter struct {
	Language     []string `json:"language"`
	Keywords     []string `json:"keywords"`
	OnlyUsers    []string `json:"only_users"`
	BlockedUsers []string `json:"blocked_users"`
	BlockedWords []string `json:"banned_words"`
}

// Twitter is a struct to hold and operate with the Twitter client, including
// using timeouts.
type Twitter struct {
	Callback func(*Tweet)

	ctx    context.Context
	filter *Filter
	client *twitter.Client
	cancel context.CancelFunc
}

// Credentials is a struct used to store and access the Twitter API keys.
type Credentials struct {
	AccessKey      string `json:"access_key"`
	ConsumerKey    string `json:"consumer_key"`
	AccessSecret   string `json:"access_secret"`
	ConsumerSecret string `json:"consumer_secret"`
}

// Stop will stop the filter process, if running.
func (t *Twitter) Stop() {
	t.cancel()
}

// Start kicks off the Twitter stream filter and receiver. This function DOES NOT block and returns an
// error of nil if successful.
func (t *Twitter) Start() error {
	s, err := t.client.Streams.Filter(
		&twitter.StreamFilterParams{
			Track:         t.filter.Keywords,
			Language:      t.filter.Language,
			StallWarnings: twitter.Bool(true),
		},
	)
	if err != nil {
		return fmt.Errorf("unable to start Twitter filter: %w", err)
	}
	d := twitter.NewSwitchDemux()
	d.Tweet = t.receive
	go func(x context.Context, e twitter.SwitchDemux, h *twitter.Stream) {
		for {
			select {
			case <-x.Done():
				h.Stop()
				return
			case m := <-h.Messages:
				e.Handle(m)
			}
		}
	}(t.ctx, d, s)
	return nil
}
func (f Filter) match(u, c string) bool {
	if len(f.BlockedUsers) > 0 {
		for i := range f.BlockedUsers {
			if strings.ToLower(f.BlockedUsers[i]) == u {
				return false
			}
		}
	}
	if len(f.BlockedWords) > 0 {
		for i := range f.BlockedWords {
			if strings.Contains(c, f.BlockedWords[i]) {
				return false
			}
		}
	}
	if len(f.OnlyUsers) > 0 {
		for i := range f.OnlyUsers {
			if strings.ToLower(f.OnlyUsers[i]) == u {
				return true
			}
		}
		return false
	}
	return true
}
func (t Twitter) receive(x *twitter.Tweet) {
	if t.filter != nil {
		if !t.filter.match(strings.ToLower(x.User.ScreenName), x.Text) {
			return
		}
	}
	r := &Tweet{
		ID:        x.ID,
		User:      x.User.Name,
		Text:      x.Text,
		UserName:  x.User.ScreenName,
		UserPhoto: x.User.ProfileImageURLHttps,
	}
	if x.Retweeted {
		if len(r.Text) > 0 {
			r.Text = fmt.Sprintf("%s\nRT @%s: %s", r.Text, x.RetweetedStatus.User.ScreenName, x.RetweetedStatus.Text)
		} else {
			r.Text = fmt.Sprintf("RT @%s: %s", x.RetweetedStatus.User.ScreenName, x.RetweetedStatus.Text)
		}
	}
	if c, err := x.CreatedAtTime(); err == nil {
		r.Time = c.Unix()
	}
	if len(x.Entities.Media) > 0 {
		r.Images = make([]string, 0, len(x.Entities.Media))
		for i := range x.Entities.Media {
			if x.Entities.Media[i].Type == "photo" {
				r.Images = append(r.Images, x.Entities.Media[i].MediaURLHttps)
			}
		}
	}
	if t.Callback != nil {
		t.Callback(r)
	}
}

// NewTwitter creates and establishes a Twitter session with the provided Access and Consumer Keys/Secrets
// and a Timeout. This function will return an error if it cannot reach Twitter or authentication failed.
func NewTwitter(x context.Context, timeout time.Duration, f *Filter, a *Credentials) (*Twitter, error) {
	if a == nil {
		return nil, ErrNoAuth
	}
	if f == nil || len(f.Keywords) == 0 {
		return nil, ErrEmptyFilter
	}
	c := oauth1.NewConfig(a.ConsumerKey, a.ConsumerSecret)
	i := c.Client(oauth1.NoContext, oauth1.NewToken(a.AccessKey, a.AccessSecret))
	i.Timeout = timeout
	t := &Twitter{
		filter: f,
		client: twitter.NewClient(i),
	}
	if _, _, err := t.client.Accounts.VerifyCredentials(nil); err != nil {
		return nil, fmt.Errorf("cannot authenticate to Twitter: %w", err)
	}
	t.ctx, t.cancel = context.WithCancel(x)
	return t, nil
}
