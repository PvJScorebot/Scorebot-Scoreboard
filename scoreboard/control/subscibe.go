// Copyright(C) 2020 iDigitalFlame
//
// This program is free software: you can redistribute it and / or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.If not, see <https://www.gnu.org/licenses/>.
//

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
	tweetBufferSize  = 2048
	clientBufferSize = 2048
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
type tweets struct {
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
	Callback    func(*game.Game)
	Subscribers map[int64]*Subscription

	log     logx.Log
	api     *web.API
	ctx     context.Context
	twitter *tweets
}

// Stop attempts to stop all WebSockets and close the connections.
func (c *Collection) Stop() {
	if len(c.Subscribers) > 0 {
		for _, v := range c.Subscribers {
			for i := range v.clients {
				v.clients[i].client.Close()
				v.clients[i] = nil
			}
			close(v.new)
		}
	}
	if c.twitter != nil {
		close(c.twitter.new)
		c.twitter = nil
	}
}
func (t *tweets) update(c *Collection) {
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
			c.log.Debug("Removed Tweet ID \"%X\" due to timeout!", t.list[i].ID)
		}
	}
	t.list = x
}
func (t *tweets) receive(x *web.Tweet) {
	x.Time = time.Now().Add(t.timeout).Unix()
	t.new <- x
}

// Sync informs the collection to download any updates and preform any maintainance
// on the clients list, including pruning clients.
func (c *Collection) Sync(t time.Duration) {
	if c.ctx.Err() != nil {
		c.Stop()
		return
	}
	x, f := context.WithTimeout(c.ctx, t)
	defer f()
	go func(z context.Context, y context.CancelFunc, i *Collection) {
		i.sync(z)
		y()
	}(x, f, c)
	<-x.Done()
	if x.Err() == context.DeadlineExceeded {
		c.log.Error("Collection Sync function ran over timeout of %s!", t.String())
	}
}
func (c *Collection) sync(x context.Context) {
	r := []int64{}
	for _, v := range c.Subscribers {
		if len(v.clients) == 0 {
			if v.tag {
				r = append(r, v.Game)
			} else {
				v.tag = true
			}
		}
		if x.Err() != nil {
			return
		}
		v.update(x, c)
	}
	if x.Err() != nil {
		return
	}
	if len(r) > 0 {
		for i := range r {
			if x.Err() != nil {
				return
			}
			c.log.Debug("Removing unused subscription for Game %d.", r[i])
			close(c.Subscribers[r[i]].new)
			delete(c.Subscribers, r[i])
		}
	}
	if c.twitter != nil && x.Err() == nil {
		c.twitter.update(c)
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
			l.Error("newclient function recovered from a panic: %s", err)
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
			new:     make(chan *web.Stream, clientBufferSize),
			last:    r,
			Game:    int64(h),
			clients: make([]*stream, 0, 1),
		}
		if c.Callback != nil {
			c.Callback(g.last)
		}
		if c.twitter != nil {
			g.last.Tweets.Tweets = c.twitter.list
		}
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
func (s *Subscription) update(x context.Context, c *Collection) {
	defer func(l logx.Log) {
		if err := recover(); err != nil {
			l.Error("update function recovered from a panic: %s", err)
		}
	}(c.log)
	if len(s.new) > 0 {
		for i := 0; len(s.new) > 0; i++ {
			s.clients = append(s.clients, &stream{ok: true, client: <-s.new})
		}
	}
	if x.Err() != nil {
		return
	}
	c.log.Debug("Checking for update for subscribed Game %d...", s.Game)
	var g *game.Game
	if err := c.api.GetJSON(fmt.Sprintf("api/scoreboard/%d/", s.Game), &g); err != nil {
		c.log.Error("Error retriving data for Game ID %d: %s", s.Game, err.Error())
		return
	}
	g.Meta.ID = s.Game
	if c.Callback != nil {
		c.Callback(g)
	}
	if c.twitter != nil {
		g.Tweets.Tweets = c.twitter.list
	}
	c.log.Debug("Running game comparison on Game %d...", s.Game)
	if x.Err() != nil {
		return
	}
	var u []*game.Update
	s.cache, u = g.Difference(s.last)
	s.last = g
	if len(u) > 0 {
		c.log.Debug("%d Updates detected in Game %d, updating clients.", len(u), s.Game)
		l := make([]*stream, 0, len(s.clients))
		for i := range s.clients {
			if x.Err() != nil || i > len(s.clients) {
				return
			}
			if s.clients[i].ok {
				s.clients[i].ok = false
				if err := s.clients[i].client.WriteJSON(u); err != nil {
					c.log.Error("Received error by client \"%s\", removing: %s", s.clients[i].client.IP(), err.Error())
					s.clients[i].client.Close()
				} else {
					s.clients[i].ok = true
					l = append(l, s.clients[i])
				}
			} else {
				s.clients[i].client.Close()
			}
		}
		s.clients = l
	}
}

// SetupTwitter creates and starts the functions to monitor Tweets.
func (c *Collection) SetupTwitter(t time.Duration) func(*web.Tweet) {
	c.twitter = &tweets{
		new:     make(chan *web.Tweet, tweetBufferSize),
		list:    make([]*web.Tweet, 0),
		timeout: t,
	}
	return c.twitter.receive
}

// NewCollection creates a collection instance from the provided logger and API.
func NewCollection(x context.Context, a *web.API, l logx.Log) *Collection {
	return &Collection{
		api:         a,
		log:         l,
		ctx:         x,
		Subscribers: make(map[int64]*Subscription),
	}
}
