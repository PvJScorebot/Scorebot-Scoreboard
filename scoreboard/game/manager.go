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

package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/PurpleSec/logx"
	"github.com/PurpleSec/parseurl"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/gorilla/websocket"
	"github.com/stvp/slug"
)

var errMissingGame = errors.New("game ID is missing from JSON data")

type hello uint64
type tweet struct {
	ID        uint64
	User      string
	Text      string
	Images    []string
	UserName  string
	UserPhoto string

	expire int64
}
type stream struct {
	ok bool
	*websocket.Conn
}
type tweets struct {
	new     chan *twitter.Tweet
	current []tweet
	timeout time.Duration
}

// Manager is a struct that contains for a map of subs and controls the connections between Scorebot
// and the Scoreboard clients.
type Manager struct {
	Games []meta

	log     logx.Log
	url     url.URL
	tick    *time.Ticker
	subs    map[uint64]*subscription
	client  *http.Client
	active  map[string]uint64
	assets  string
	running uint32
	twitter *tweets
	timeout time.Duration
}
type subscription struct {
	ID uint64

	new     chan *websocket.Conn
	last    game
	stale   uint32
	cache   []update
	clients []*stream
}

func init() {
	slug.Replacement = '-'
}
func (m *Manager) close() {
	for n, s := range m.subs {
		for i := range s.clients {
			s.clients[i].Close()
			s.clients[i] = nil
		}
		close(s.new)
		delete(m.subs, n)
	}
	if m.twitter != nil {
		m.twitter.current = nil
	}
	m.tick.Stop()
}
func (t tweet) Sum() uint64 {
	return t.ID
}

// Game will attempt to resolve the game name provided to an active game ID. This function will replace
// any spaces and invalid characters and matches the Game name without case sensitivity. THis function returns
// zero if no Game was found
func (m *Manager) Game(s string) uint64 {
	return m.active[strings.ToLower(slug.Clean(s))]
}

// New attempts to add the supplied web client to the Subscription swarm.
func (m *Manager) New(n *websocket.Conn) {
	defer func(l logx.Log) {
		if err := recover(); err != nil {
			l.Error("Collection newclient function recovered from a panic: %s!", err)
		}
	}(m.log)
	m.log.Debug("Received a connection from %q, listening for Hello...", n.RemoteAddr().String())
	var h hello
	if err := n.ReadJSON(&h); err != nil {
		m.log.Error("Could not read Hello message from %q, closing: %s!", n.RemoteAddr().String(), err.Error())
		n.Close()
		return
	}
	m.log.Debug("Received Hello with requested Game ID %d from %q.", h, n.RemoteAddr().String())
	s, ok := m.subs[uint64(h)]
	if !ok || s == nil {
		m.log.Debug("Checking Game ID %d, requested by %q...", h, n.RemoteAddr().String())
		var g game
		if err := m.getJSON(context.Background(), fmt.Sprintf("api/scoreboard/%d/", h), &g); err != nil {
			m.log.Error("Error retriving data for Game ID %d: %s!", h, err.Error())
			n.Close()
			return
		}
		if len(g.Meta.Name) == 0 && len(g.Teams) == 0 {
			m.log.Error("Game ID %d is empty, ignoring!", h)
			n.Close()
			return
		}
		g.Meta.ID = uint64(h)
		for i := range m.Games {
			if m.Games[i].ID == g.Meta.ID {
				g.Meta.End = m.Games[i].End
				g.Meta.Start = m.Games[i].Start
				g.Meta.Status = m.Games[i].Status
				break
			}
		}
		s = &subscription{
			ID:      g.Meta.ID,
			new:     make(chan *websocket.Conn, 128),
			last:    g,
			clients: make([]*stream, 0, 1),
		}
		if m.twitter != nil {
			s.last.Tweets = m.twitter.current
		}
		s.cache, _ = s.last.Delta(m.assets, nil)
		m.subs[g.Meta.ID] = s
	}
	atomic.StoreUint32(&s.stale, 0)
	n.WriteJSON(s.cache)
	s.new <- n
}

// Start will start the Manager content thread. This function takes a context that will be used
// to stop and cancel all running processes.
func (m *Manager) Start(x context.Context) {
	for {
		select {
		case <-x.Done():
			m.close()
			return
		case <-m.tick.C:
			if atomic.LoadUint32(&m.running) == 0 {
				go m.startUpdate(x)
			}
		}
	}
}
func (m *Manager) update(x context.Context) {
	m.log.Trace("Starting update...")
	if err := m.getJSON(x, "api/games/", &m.Games); err != nil {
		m.log.Error("Error occurred during update tick: %s", err.Error())
		return
	}
	for i := range m.Games {
		n := slug.Clean(m.Games[i].Name)
		if !m.Games[i].Active() {
			delete(m.active, n)
			continue
		}
		if _, ok := m.active[n]; !ok {
			m.active[n] = m.Games[i].ID
			m.log.Debug("Added Game name mapping %q to ID %d.", n, m.Games[i].ID)
		}
	}
	select {
	case <-x.Done():
		return
	default:
		break
	}
	var r []uint64
	for _, s := range m.subs {
		if len(s.clients) == 0 {
			if atomic.LoadUint32(&s.stale) == 1 {
				r = append(r, s.ID)
				continue
			}
			atomic.StoreUint32(&s.stale, 1)
		}
		select {
		case <-x.Done():
			return
		default:
		}
		s.update(x, m)
	}
	for i := range r {
		select {
		case <-x.Done():
			return
		default:
		}
		m.log.Debug("Removing unused subscription for Game %d.", r[i])
		close(m.subs[r[i]].new)
		delete(m.subs, r[i])
	}
	if m.twitter != nil {
		m.twitter.update(x, m)
	}
	m.log.Debug("Read %d Games from scorebot, update finished.", len(m.Games))
}
func (h *hello) UnmarshalJSON(b []byte) error {
	var m map[string]uint64
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	v, ok := m["game"]
	if !ok {
		return errMissingGame
	}
	*h = hello(v)
	return nil
}
func (m *Manager) startUpdate(x context.Context) {
	atomic.StoreUint32(&m.running, 1)
	c, f := context.WithTimeout(x, m.timeout)
	go func(y context.Context, w context.CancelFunc, q *Manager) {
		defer func() {
			if err := recover(); err != nil {
				q.log.Error("Panic occurred during manager tick: %s!", err)
				w()
			}
		}()
		q.update(y)
		w()
	}(c, f, m)
	<-c.Done()
	if c.Err() == context.DeadlineExceeded {
		m.log.Warning("Collection update function ran over timeout of %s!", m.timeout.String())
	}
	f()
	atomic.StoreUint32(&m.running, 0)
}
func (t *tweets) update(x context.Context, m *Manager) {
	var (
		n = time.Now().Unix()
		c = make([]tweet, 0, len(t.current))
	)
	for len(t.new) > 0 {
		select {
		case <-x.Done():
			return
		default:
		}
		var (
			x = <-t.new
			r = tweet{
				ID:        uint64(x.ID),
				User:      x.User.Name,
				Text:      x.Text,
				expire:    n + int64(t.timeout.Seconds()),
				UserName:  x.User.ScreenName,
				UserPhoto: x.User.ProfileImageURLHttps,
			}
		)
		if x.Retweeted {
			if len(r.Text) > 0 {
				r.Text = fmt.Sprintf("%s\nRT @%s: %s", r.Text, x.RetweetedStatus.User.ScreenName, x.RetweetedStatus.Text)
			} else {
				r.Text = fmt.Sprintf("RT @%s: %s", x.RetweetedStatus.User.ScreenName, x.RetweetedStatus.Text)
			}
		}
		if len(x.Entities.Media) > 0 {
			r.Images = make([]string, 0, len(x.Entities.Media))
			for i := range x.Entities.Media {
				if x.Entities.Media[i].Type != "photo" {
					continue
				}
				r.Images = append(r.Images, x.Entities.Media[i].MediaURLHttps)
			}
		}
		c = append(c, r)
	}
	for i := range t.current {
		select {
		case <-x.Done():
			return
		default:
		}
		if t.current[i].expire > n {
			c = append(c, t.current[i])
		}
		m.log.Debug("Removed Tweet ID \"%X\" due to timeout!", t.current[i].ID)
	}
	t.current = c
}
func (s *subscription) update(x context.Context, m *Manager) {
	defer func(l logx.Log) {
		if err := recover(); err != nil {
			l.Error("Game subscription update function recovered from a panic: %s!", err)
		}
	}(m.log)
	for len(s.new) > 0 {
		s.clients = append(s.clients, &stream{true, <-s.new})
	}
	select {
	case <-x.Done():
		return
	default:
	}
	m.log.Debug("Checking for update for subscribed Game %d...", s.ID)
	var g game
	if err := m.getJSON(x, fmt.Sprintf("api/scoreboard/%d/", s.ID), &g); err != nil {
		m.log.Error("Error retriving data for Game ID %d: %s!", s.ID, err.Error())
		return
	}
	g.Meta.ID = s.ID
	for i := range m.Games {
		if m.Games[i].ID == g.Meta.ID {
			g.Meta.End = m.Games[i].End
			g.Meta.Start = m.Games[i].Start
			g.Meta.Status = m.Games[i].Status
			break
		}
	}
	if m.twitter != nil {
		g.Tweets = m.twitter.current
	}
	select {
	case <-x.Done():
		return
	default:
	}
	var u []update
	m.log.Debug("Running game comparison on Game %d...", s.ID)
	s.cache, u = g.Delta(m.assets, &s.last)
	s.last = g
	if len(u) > 0 {
		m.log.Debug("%d Updates detected in Game %d, updating clients...", len(u), s.ID)
		r := make([]*stream, 0, len(s.clients))
		for i := range s.clients {
			select {
			case <-x.Done():
				return
			default:
			}
			if i > len(s.clients) {
				return
			}
			if !s.clients[i].ok {
				s.clients[i].Close()
				continue
			}
			s.clients[i].ok = false
			if err := s.clients[i].WriteJSON(u); err != nil {
				m.log.Error("Received error by client %q, removing: %s!", s.clients[i].RemoteAddr().String(), err.Error())
				s.clients[i].Close()
				continue
			}
			s.clients[i].ok = true
			r = append(r, s.clients[i])
		}
		s.clients = r
	}
}

// Twitter creates and returns the Twitter channel. This channel can be used to submit Tweets
// to be sent to the scoreboard.
func (m *Manager) Twitter(t time.Duration) chan *twitter.Tweet {
	m.twitter = &tweets{new: make(chan *twitter.Tweet), timeout: t}
	return m.twitter.new
}
func (m Manager) get(x context.Context, u string) ([]byte, error) {
	m.url.Path = fmt.Sprintf("%s/", path.Join(m.url.Path, u))
	var (
		c, f   = context.WithTimeout(x, m.timeout)
		r, err = http.NewRequestWithContext(c, http.MethodGet, m.url.String(), nil)
	)
	defer f()
	if err != nil {
		return nil, err
	}
	o, err := m.client.Do(r)
	if err != nil {
		return nil, err
	}
	if o.Body == nil {
		return nil, fmt.Errorf("request %q returned an empty body", m.url.String())
	}
	defer o.Body.Close()
	if o.StatusCode >= 400 {
		return nil, fmt.Errorf("request %q returned status code %d", m.url.String(), o.StatusCode)
	}
	b, err := ioutil.ReadAll(o.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading from URL %q: %w", m.url.String(), err)
	}
	return b, nil
}
func (m Manager) getJSON(x context.Context, u string, o interface{}) error {
	r, err := m.get(x, u)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(r, &o); err != nil {
		return fmt.Errorf("unable to unmarshal JSON: %w", err)
	}
	return nil
}

// New creates a collection instance from the provided logger, timeout and API URL endpoint.
func New(burl, d string, tick, t time.Duration, l logx.Log) (*Manager, error) {
	u, err := parseurl.Parse(burl)
	if err != nil {
		return nil, fmt.Errorf("could not unpack provided URL %q: %w", burl, err)
	}
	if !u.IsAbs() {
		u.Scheme = "http"
	}
	if len(d) == 0 {
		d = u.String()
	}
	m := &Manager{
		log:    l,
		url:    *u,
		subs:   make(map[uint64]*subscription),
		tick:   time.NewTicker(tick),
		active: make(map[string]uint64),
		assets: d,
		client: &http.Client{
			Timeout: t,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           (&net.Dialer{Timeout: t, KeepAlive: t, DualStack: true}).DialContext,
				IdleConnTimeout:       t,
				TLSHandshakeTimeout:   t,
				ExpectContinueTimeout: t,
				ResponseHeaderTimeout: t,
			},
		},
		timeout: t,
	}
	return m, nil
}
