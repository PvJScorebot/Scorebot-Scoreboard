package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/iDigitalFlame/scoreboard/web"
	"github.com/iDigitalFlame/scoreboard/control/game"
)

var (
	// ErrMissingGame is returned when attempting to Unmarshal a Hello struct that does not
	// contain a Game ID mapping.
	ErrMissingGame = errors.New("game ID is missing from JSON data")
)

type hello int64
type logger interface {
	Info(string, ...interface{})
	Debug(string, ...interface{})
	Error(string, ...interface{})
}
type mapper struct {
	add, current []*web.Tweet
}

// Subscription is a collection of Clients that have subscripted to a specific
// Game ID.
type Subscription struct {
	Game int64

	tag     bool
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
	tweets  *mapper
	timeout time.Duration
}

// Sync informs the collection to download any updates and preform any maintainance
// on the clients list, including pruning clients.
func (c *Collection) Sync() {
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
		for _, x := range r {
			delete(c.Subscribers, x)
		}
	}
	if c.tweets != nil {
		c.expireTweets()
	}
}

// Stop attempts to stop all WebSockets and close the connections.
func (c *Collection) Stop() error {
	var err error
	for _, v := range c.Subscribers {
		for i := range v.clients {
			err = v.clients[i].Close()
		}
	}
	return err
}
func (c *Collection) expireTweets() {
	if len(c.tweets.current) == 0 && len(c.tweets.add) == 0 {
		return
	}
	n := time.Now().Unix()
	nl := make([]*web.Tweet, 0, len(c.tweets.current)+len(c.tweets.add))
	for i := range c.tweets.current {
		if c.tweets.current[i].Time > n {
			nl = append(nl, c.tweets.current[i])
		} else {
			c.log.Debug("Removed Tweet ID \"%d\" due to timeout!", c.tweets.current[i].ID)
		}
	}
	if len(c.tweets.add) > 0 {
		nl = append(nl, c.tweets.add...)
		c.tweets.add = c.tweets.add[0:0]
	}
	c.tweets.current = nl
}
func (c *Collection) receive(t *web.Tweet) {
	t.Time = time.Now().Add(c.timeout).Unix()
	c.tweets.add = append(c.tweets.add, t)
}
func (s *Subscription) update(c *Collection) {
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
	if c.tweets != nil {
		g.Tweets.Tweets = c.tweets.current
	}
	g.GenerateHash()
	c.log.Debug("Running game comparison on Game \"%d\"..", s.Game)
	n, u := g.Difference(s.last)
	s.last = g
	s.cache = n
	if len(u) > 0 {
		c.log.Debug("%d Updates detected in Game \"%d\", updating clients..", len(u), s.Game)
		x := s.clients[:0]
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
	c.log.Debug("Received a connection from \"%s\", listening for Hello..", n.IP())
	var h hello
	if err := n.ReadJSON(&h); err != nil {
		c.log.Error("Could not read Hello message from \"%s\", (%s) closing!", n.IP(), err.Error())
		n.Close()
		return
	}
	c.log.Debug("Received Hello with requested Game ID \"%d\" from \"%s\".", h, n.IP())
	g, ok := c.Subscribers[int64(h)]
	if !ok {
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
		g = &Subscription{Game: int64(h), last: r, clients: make([]*web.Stream, 0)}
		if c.gcb != nil {
			c.gcb(g.last)
		}
		if c.tweets != nil {
			g.last.Tweets.Tweets = c.tweets.current
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
	s.clients = append(s.clients, n)
	n.WriteJSON(s.cache)
}

// NewCollection creates a collection instance from the provded logger and API.
func NewCollection(a *web.API, l logger) *Collection {
	return &Collection{Subscribers: make(map[int64]*Subscription), api: a, log: l}
}

// GameCallback sets the callback function triggered on a received game.
func (c *Collection) GameCallback(f func(*game.Game)) {
	c.gcb = f
}

// SetupTwitter creates and starts the functions to monitor Tweets.
func (c *Collection) SetupTwitter(t time.Duration) func(*web.Tweet) {
	c.tweets = &mapper{add: make([]*web.Tweet, 0), current: make([]*web.Tweet, 0)}
	c.timeout = t
	return c.receive
}
