package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/iDigitalFlame/logx/logx"
	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/control/game"
	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/web"
)

const (
	// TweetBufferSize is the size of the incoming tweets chan buffer
	TweetBufferSize = 2048
	// ClientBufferSize is the size of the incoming clients chan buffer.
	ClientBufferSize = 2048
)

var (
	// ErrMissingGame is returned when attempting to Unmarshal a Hello struct that does not
	// contain a Game ID mapping.
	ErrMissingGame = errors.New("game ID is missing from JSON data")
)

type hello int64
type stream struct {
	ok     bool
	client *web.Stream
}
type tweetbuf struct {
	new     chan *web.Tweet
	list    []*web.Tweet
	timeout time.Duration
}

// Subscription is a collection of Clients that have subscripted to a specific
// Game ID.
type Subscription struct {
	Game int64

	tag     bool
	new     chan *web.Stream
	last    *game.Game
	cache   []*game.Update
	clients []*stream
}

// Collection is a struct that contains for a map of subscribers.
type Collection struct {
	Subscribers map[int64]*Subscription

	log     logx.Log
	api     *web.API
	gcb     func(*game.Game)
	twitter *tweetbuf
}

// Stop attempts to stop all WebSockets and close the connections.
func (c *Collection) Stop() error {
	var err error
	for _, v := range c.Subscribers {
		for i := range v.clients {
			err = v.clients[i].client.Close()
			v.clients[i] = nil
		}
		close(v.new)
	}
	if c.twitter != nil {
		close(c.twitter.new)
	}
	return err
}
func (t *tweetbuf) update(c *Collection) {
	if len(t.new) > 0 {
		for i := 0; len(t.new) > 0; i++ {
			t.list = append(t.list, <-t.new)
		}
	}
	if len(t.list) == 0 {
		return
	}
	n := time.Now().Unix()
	x := make([]*web.Tweet, 0, len(t.list))
	for i := range t.list {
		if t.list[i].Time > n {
			x = append(x, t.list[i])
		} else {
			c.log.Debug("Removed Tweet ID \"%d\" due to timeout!", t.list[i].ID)
		}
	}
	t.list = x
}
func (t *tweetbuf) receive(x *web.Tweet) {
	x.Time = time.Now().Add(t.timeout).Unix()
	t.new <- x
}

// Sync informs the collection to download any updates and preform any maintainance
// on the clients list, including pruning clients.
func (c *Collection) Sync(t time.Duration) {
	x, f := context.WithTimeout(context.Background(), t)
	defer f()
	go func(z context.Context, i context.CancelFunc, o *Collection) {
		o.doSync(z)
		i()
	}(x, f, c)
	<-x.Done()
	if x.Err() == context.DeadlineExceeded {
		c.log.Error("Collection Sync function ran over timeout of %s!", t.String())
	}
}
func (h *hello) UnmarshalJSON(b []byte) error {
	var m map[string]int64
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	v, ok := m["game"]
	if !ok {
		return ErrMissingGame
	}
	*h = hello(v)
	return nil
}

// NewClient attempts to add the client 'n' to the Subscription swarm.
func (c *Collection) NewClient(n *web.Stream) {
	defer func(l logx.Log) {
		if err := recover(); err != nil {
			l.Error("newclient gofunc: recovered from a panic: %s", err)
		}
	}(c.log)
	c.log.Debug("Received a connection from \"%s\", listening for Hello.", n.IP())
	var h hello
	if err := n.ReadJSON(&h); err != nil {
		c.log.Error("Could not read Hello message from \"%s\", closing: %s", n.IP(), err.Error())
		n.Close()
		return
	}
	c.log.Debug("Received Hello with requested Game ID %d from \"%s\".", h, n.IP())
	g, ok := c.Subscribers[int64(h)]
	if !ok || g == nil {
		c.log.Debug("Checking Game ID %d, requested by \"%s\".", h, n.IP())
		var r *game.Game
		if err := c.api.GetJSON(fmt.Sprintf("api/scoreboard/%d/", h), &r); err != nil {
			c.log.Error("Error retriving data for Game ID %d: %s", h, err.Error())
			n.Close()
			return
		}
		if len(r.Meta.Name) == 0 && len(r.Teams) == 0 {
			c.log.Error("Game ID %d is empty, ignoring!", h)
			n.Close()
			return
		}
		r.Meta.ID = int64(h)
		g = &Subscription{
			new:     make(chan *web.Stream, ClientBufferSize),
			last:    r,
			Game:    int64(h),
			clients: make([]*stream, 0, 1),
		}
		if c.gcb != nil {
			c.gcb(g.last)
		}
		if c.twitter != nil {
			g.last.Tweets.Tweets = c.twitter.list
		}
		g.last.GenerateHash()
		g.cache, _ = g.last.Difference(nil)
		c.Subscribers[int64(h)] = g
	}
	g.NewClient(n)
}
func (c *Collection) doSync(z context.Context) {
	r := []int64{}
	for _, v := range c.Subscribers {
		if len(v.clients) == 0 {
			if v.tag {
				r = append(r, v.Game)
			} else {
				v.tag = true
			}
		}
		if z.Err() != nil {
			return
		}
		v.update(z, c)
	}
	if z.Err() != nil {
		return
	}
	if len(r) > 0 {
		for i := range r {
			if z.Err() != nil {
				return
			}
			c.log.Debug("Removing unused subscription for Game %d.", r[i])
			close(c.Subscribers[r[i]].new)
			delete(c.Subscribers, r[i])
		}
	}
	if c.twitter != nil && z.Err() == nil {
		c.twitter.update(c)
	}
}

// NewClient adds the client 'n' to this subscription.
func (s *Subscription) NewClient(n *web.Stream) {
	s.tag = false
	n.WriteJSON(s.cache)
	s.new <- n
}

// GameCallback sets the callback function triggered on a received game.
func (c *Collection) GameCallback(f func(*game.Game)) {
	c.gcb = f
}

// NewCollection creates a collection instance from the provided logger and API.
func NewCollection(a *web.API, l logx.Log) *Collection {
	return &Collection{
		api:         a,
		log:         l,
		Subscribers: make(map[int64]*Subscription),
	}
}
func (s *Subscription) update(z context.Context, c *Collection) {
	if len(s.new) > 0 {
		for i := 0; len(s.new) > 0; i++ {
			s.clients = append(s.clients, &stream{ok: true, client: <-s.new})
		}
	}
	c.log.Debug("Checking for update for subscribed Game %d...", s.Game)
	var g *game.Game
	if err := c.api.GetJSON(fmt.Sprintf("api/scoreboard/%d/", s.Game), &g); err != nil {
		c.log.Error("Error retriving data for Game ID %d: %s", s.Game, err.Error())
		return
	}
	g.Meta.ID = s.Game
	if c.gcb != nil {
		c.gcb(g)
	}
	if c.twitter != nil {
		g.Tweets.Tweets = c.twitter.list
	}
	g.GenerateHash()
	c.log.Debug("Running game comparison on Game %d...", s.Game)
	if z.Err() != nil {
		return
	}
	n, u := g.Difference(s.last)
	s.last = g
	s.cache = n
	if len(u) > 0 {
		c.log.Debug("%d Updates detected in Game %d, updating clients.", len(u), s.Game)
		x := make([]*stream, 0, len(s.clients))
		for i := range s.clients {
			if z.Err() != nil || i > len(s.clients) {
				return
			}
			if s.clients[i].ok {
				s.clients[i].ok = false
				if err := s.clients[i].client.WriteJSON(u); err != nil {
					c.log.Error("Received error by client \"%s\", removing: %s", s.clients[i].client.IP(), err.Error())
					s.clients[i].client.Close()
				} else {
					s.clients[i].ok = true
					x = append(x, s.clients[i])
				}
			} else {
				s.clients[i].client.Close()
			}
		}
		s.clients = x
	}
}

// SetupTwitter creates and starts the functions to monitor Tweets.
func (c *Collection) SetupTwitter(t time.Duration) func(*web.Tweet) {
	c.twitter = &tweetbuf{
		new:     make(chan *web.Tweet, TweetBufferSize),
		list:    make([]*web.Tweet, 0),
		timeout: t,
	}
	return c.twitter.receive
}
