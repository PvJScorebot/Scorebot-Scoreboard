package control

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/control/game"
	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/web"
	"golang.org/x/xerrors"
)

const (
	// TweetBufferSize is the size of the incomming tweets chan buffer
	TweetBufferSize = 2048
	// ClientBufferSize is the size of the incomming clients chan buffer.
	ClientBufferSize = 2048
)

var (
	// ErrMissingGame is returned when attempting to Unmarshal a Hello struct that does not
	// contain a Game ID mapping.
	ErrMissingGame = xerrors.New("game ID is missing from JSON data")
)

type hello int64
type logger interface {
	Info(string, ...interface{})
	Debug(string, ...interface{})
	Error(string, ...interface{})
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
	clients []*web.Stream
}

// Collection is a struct that contains for a map of subscrbers.
type Collection struct {
	Subscribers map[int64]*Subscription

	log     logger
	api     *web.API
	gcb     func(*game.Game)
	twitter *tweetbuf
}

func (t *tweetbuf) addNew() {
	for {
		select {
		case n := <-t.new:
			t.list = append(t.list, n)
		default:
			return
		}
	}
}
func (c *Collection) tsync() {
	r := []int64{}
	for _, v := range c.Subscribers {
		if len(v.clients) == 0 {
			if v.tag {
				r = append(r, v.Game)
			} else {
				v.tag = true
			}
		}
		v.update(c)
	}
	if len(r) > 0 {
		for i := range r {
			c.log.Debug("Removing unused subscription for Game \"%d\"..", r[i])
			close(c.Subscribers[r[i]].new)
			delete(c.Subscribers, r[i])
		}
	}
	if c.twitter != nil {
		c.twitter.update(c)
	}
}
func (s *Subscription) addNew() {
	for {
		select {
		case n := <-s.new:
			s.clients = append(s.clients, n)
		default:
			return
		}
	}
}

// Stop attempts to stop all WebSockets and close the connections.
func (c *Collection) Stop() error {
	var err error
	for _, v := range c.Subscribers {
		for i := range v.clients {
			err = v.clients[i].Close()
			v.clients[i] = nil
		}
	}
	if c.twitter != nil {
		close(c.twitter.new)
	}
	return err
}
func (t *tweetbuf) update(c *Collection) {
	t.addNew()
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
	go func(o *Collection, i context.CancelFunc) {
		o.tsync()
	}(c, f)
	<-x.Done()
	if x.Err() == context.DeadlineExceeded {
		c.log.Error("Collection Sync function ran over timeout of %s!", t.String())
	}
}
func (s *Subscription) update(c *Collection) {
	s.addNew()
	c.log.Debug("Checking for update for subscribed Game \"%d\"..", s.Game)
	var g *game.Game
	if err := c.api.GetJSON(fmt.Sprintf("api/scoreboard/%d/", s.Game), &g); err != nil {
		c.log.Error("Error retriving data for Game ID \"%d\": %s", s.Game, err.Error())
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
	c.log.Debug("Running game comparison on Game \"%d\"..", s.Game)
	n, u := g.Difference(s.last)
	s.last = g
	s.cache = n
	if len(u) > 0 {
		c.log.Debug("%d Updates detected in Game \"%d\", updating clients..", len(u), s.Game)
		x := make([]*web.Stream, 0, len(s.clients))
		for i := range s.clients {
			if err := s.clients[i].WriteJSON(u); err != nil {
				c.log.Error("Received error by client \"%s\", removing: %s", s.clients[i].IP(), err.Error())
			} else {
				x = append(x, s.clients[i])
			}
		}
		s.clients = x
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
	defer func(l logger) {
		if err := recover(); err != nil {
			l.Error("http gofunc: ecovered from a panic: %s", err)
		}
	}(c.log)
	c.log.Debug("Received a connection from \"%s\", listening for Hello..", n.IP())
	var h hello
	if err := n.ReadJSON(&h); err != nil {
		c.log.Error("Could not read Hello message from \"%s\", (%s) closing!", n.IP(), err.Error())
		n.Close()
		return
	}
	c.log.Debug("Received Hello with requested Game ID \"%d\" from \"%s\".", h, n.IP())
	g, ok := c.Subscribers[int64(h)]
	if !ok || g == nil {
		c.log.Debug("Checking Game ID \"%d\", requested by \"%s\"..", h, n.IP())
		var r *game.Game
		if err := c.api.GetJSON(fmt.Sprintf("api/scoreboard/%d/", h), &r); err != nil {
			c.log.Error("Error retriving data for Game ID \"%d\": %s", h, err.Error())
			n.Close()
			return
		}
		if len(r.Meta.Name) == 0 && len(r.Teams) == 0 {
			c.log.Error("Game ID \"%d\" is empty, ignoring!", h)
			n.Close()
			return
		}
		r.Meta.ID = int64(h)
		g = &Subscription{
			new:     make(chan *web.Stream, ClientBufferSize),
			last:    r,
			Game:    int64(h),
			clients: make([]*web.Stream, 0, 1),
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

// NewClient adds the client 'n' to this subscription.
func (s *Subscription) NewClient(n *web.Stream) {
	s.tag = false
	n.WriteJSON(s.cache)
	s.new <- n
}

// NewCollection creates a collection instance from the provded logger and API.
func NewCollection(a *web.API, l logger) *Collection {
	return &Collection{
		api:         a,
		log:         l,
		Subscribers: make(map[int64]*Subscription),
	}
}

// GameCallback sets the callback function triggered on a received game.
func (c *Collection) GameCallback(f func(*game.Game)) {
	c.gcb = f
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
