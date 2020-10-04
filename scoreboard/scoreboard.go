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

package scoreboard

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/PurpleSec/logx"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/gobuffalo/packr/v2"
	"github.com/gorilla/websocket"
	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/game"
)

const (
	usage = `Scorebot Scoreboard

Leaving any of the required Twitter options empty in command
line or config will result in Twitter functionality being disabled.
Required Twitter options: 'Consumer Key and Secret', 'Access Key and Secret',
'Twitter Keywords' and 'Twitter Language'.

Usage of scoreboard:
  -c <file>                 Scorebot configuration file path.
  -d                        Print default configuration and exit.
  -sbe <url>                Scorebot core address or URL (Required without "-c").
  -assets <dir>             Scoreboard secondary assets override URL.
  -dir <directory>          Scoreboard HTML override directory path.
  -log <file>               Scoreboard log file path.
  -log-level <number [0-5]> Scoreboard logging level (Default 2).
  -tick <seconds>           Scorebot poll tate, in seconds (Default 5).
  -timeout <seconds>        Scoreboard request timeout, in seconds (Default 10).
  -bind <socket>            Address and port to listen on (Default "0.0.0.0:8080").
  -cert <file>              Path to TLS certificate file.
  -key <file>               Path to TLS key file.
  -tw-ck <key>              Twitter Consumer API key.
  -tw-cs <secret>           Twitter Consumer API secret.
  -tw-ak <key>              Twitter Access API key.
  -tw-as <secret>           Twitter Access API secret.
  -tw-keywords <list>       Twitter search keywords (Comma separated)
  -tw-lang <list>           Twitter search language (Comma separated)
  -tw-expire <seconds>      Tweet display time, in seconds (Default 45).
  -tw-block-words <list>    Twitter blocked words (Comma separated).
  -tw-block-user <list>     Twitter blocked Usernames (Comma separated).
  -tw-only-users <list>     Twitter whitelisted Usernames (Comma separated).

Copyright (C) 2020 iDigitalFlame

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

`
	version = 2.2
)

var resources = packr.New("html", "../html")

type display struct {
	Game    uint64
	Twitter bool
}
type errorval struct {
	e error
	s string
}

// Scoreboard is a struct that represents the Scoreboard multiplexer. This struct is used to gather and
// compare Game data to push to Scoreboard clients.
type Scoreboard struct {
	fs     http.Handler
	ws     *websocket.Upgrader
	log    logx.Log
	dir    http.FileSystem
	key    string
	cert   string
	feed   *twitter.Stream
	html   *template.Template
	filter filter
	expire time.Duration
	*game.Manager
	*http.Server
}

// Run begins the listening process for the Scoreboard and the Game ticking threads. This
// function blocks until interrupted. This function watches the SIGINT, SIGHUP, SIGTERM and SIGQUIT
// signals and will automatically close and clean up after a signal is received.
func (s *Scoreboard) Run() error {
	return s.RunContext(context.Background())
}
func (e errorval) Error() string {
	if e.e == nil {
		return e.s
	}
	return e.s + ": " + e.e.Error()
}
func (e errorval) Unwrap() error {
	return e.e
}

// New creates a new scoreboard instance from the provided Config struct. Any errors during setup will be returned.
func (c Config) New() (*Scoreboard, error) {
	if err := c.verify(); err != nil {
		return nil, err
	}
	var (
		t    = time.Second * time.Duration(c.Timeout)
		err  error
		x, p string
	)
	if len(c.Directory) > 0 {
		p = filepath.Join(c.Directory, "public")
		d, err := os.Stat(p)
		if err != nil {
			return nil, &errorval{s: `public directory "` + p + `" does not exist`, e: err}
		}
		if !d.IsDir() {
			return nil, &errorval{s: `public directory "` + p + `" is not a directory`}
		}
		x = filepath.Join(c.Directory, "template")
	}
	var s Scoreboard
	if len(c.Log.File) > 0 {
		f, err := logx.File(c.Log.File, logx.Level(c.Log.Level))
		if err != nil {
			return nil, &errorval{s: `unable to create log file "` + c.Log.File + `"`, e: err}
		}
		s.log = logx.Multiple(f, logx.Console(logx.Level(c.Log.Level)))
	} else {
		s.log = logx.Console(logx.Level(c.Log.Level))
	}
	s.html = template.New("base")
	if err = getTemplate(s.html, x, "home.html"); err != nil {
		return nil, &errorval{s: "unable to load home template", e: err}
	}
	if err = getTemplate(s.html, x, "scoreboard.html"); err != nil {
		return nil, &errorval{s: "unable to load scoreboard template", e: err}
	}
	if s.Manager, err = game.New(c.Scorebot, c.Assets, time.Duration(c.Tick)*time.Second, t, s.log); err != nil {
		return nil, &errorval{s: "unable to setup game manager", e: err}
	}
	s.Server = &http.Server{
		Addr:              c.Listen,
		Handler:           new(http.ServeMux),
		ReadTimeout:       t,
		IdleTimeout:       t,
		WriteTimeout:      t,
		ReadHeaderTimeout: t,
	}
	s.ws = &websocket.Upgrader{
		CheckOrigin:      func(_ *http.Request) bool { return true },
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: t,
	}
	if c.twitter {
		y := twitter.NewClient(
			oauth1.NewConfig(c.Twitter.Credentials.ConsumerKey, c.Twitter.Credentials.ConsumerSecret).Client(
				context.Background(),
				oauth1.NewToken(c.Twitter.Credentials.AccessKey, c.Twitter.Credentials.AccessSecret),
			),
		)
		if _, _, err := y.Accounts.VerifyCredentials(nil); err != nil {
			return nil, &errorval{s: "cannot authenticate to Twitter: %w", e: err}
		}
		s.feed, err = y.Streams.Filter(
			&twitter.StreamFilterParams{
				Track:         c.Twitter.Filter.Keywords,
				Language:      c.Twitter.Filter.Language,
				StallWarnings: twitter.Bool(true),
			},
		)
		if err != nil {
			return nil, &errorval{s: "unable to start Twitter filter", e: err}
		}
		s.filter, s.expire = c.Twitter.Filter, time.Duration(c.Twitter.Expire)*time.Second
		s.log.Info("Twitter setup successful!")
	} else {
		s.log.Warning("Missing Twitter keys and/or filter parameters, skipping Twitter setup!")
	}
	s.key, s.cert = c.Key, c.Cert
	s.fs, s.dir = http.FileServer(&s), http.Dir(p)
	s.Server.Handler.(*http.ServeMux).HandleFunc("/", s.http)
	s.Server.Handler.(*http.ServeMux).HandleFunc("/w", s.httpWebsocket)
	return &s, nil
}
func (s *Scoreboard) startTwitter(x context.Context) {
	for c := s.Manager.Twitter(s.expire); ; {
		select {
		case <-x.Done():
			close(c)
			s.feed.Stop()
			return
		case n := <-s.feed.Messages:
			switch t := n.(type) {
			case *twitter.Tweet:
				c <- t
			case *twitter.Event:
			case *twitter.FriendsList:
			case *twitter.UserWithheld:
			case *twitter.DirectMessage:
			case *twitter.StatusDeletion:
			case *twitter.StatusWithheld:
			case *twitter.LocationDeletion:
			case *twitter.StreamLimit:
				s.log.Warning("Twitter stream thread received a StreamLimit message of %d!", t.Track)
			case *twitter.StallWarning:
				s.log.Warning("Twitter stream thread received a StallWarning message: %s!", t.Message)
			case *twitter.StreamDisconnect:
				s.log.Error("Twitter stream thread received a StreamDisconnect message: %s!", t.Reason)
				return
			case *url.Error:
				s.log.Error("Twitter stream thread received an error: %s!", t.Error())
				return
			default:
				if t != nil {
					s.log.Warning("Twitter stream thread received an unrecognized message (%T): %s\n", t, t)
				}
			}
		}
	}
}

// Open satisfies the http.FileSystem interface. This function is used to mask the packed resources and
// use any replacement files (if they exist).
func (s Scoreboard) Open(n string) (http.File, error) {
	f, err := s.dir.Open(n)
	if err == nil {
		return f, nil
	}
	return resources.Open(path.Join("public", n))
}
func getTemplate(t *template.Template, d, f string) error {
	if len(d) > 0 {
		s := filepath.Join(d, f)
		if i, err := os.Stat(s); err == nil && !i.IsDir() {
			if _, err = t.New(f).ParseFiles(s); err != nil {
				return &errorval{s: `unable to parse template "` + f + `"`, e: err}
			}
			return nil
		}
	}
	c, err := resources.FindString("template/" + f)
	if err != nil {
		return &errorval{s: `could not find template "` + f + `"`, e: err}
	}
	if _, err := t.New(f).Parse(c); err != nil {
		return &errorval{s: `unable to parse scorebot template "` + f + `"`, e: err}
	}
	return nil
}

// RunContext begins the listening process for the Scoreboard and the Game ticking threads. This
// function blocks until interrupted. This function watches the SIGINT, SIGTERM and SIGQUIT signals and will
// automatically close and clean up after a signal is received. This function accepts a Context to allow for control
// of when the Scoreboard stops without using signals.
func (s *Scoreboard) RunContext(ctx context.Context) error {
	var (
		e    = make(chan error, 1)
		w    = make(chan os.Signal, 1)
		x, c = context.WithCancel(ctx)
	)
	signal.Notify(w, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	s.log.Info("Starting Scoreboard service...")
	if len(s.cert) > 0 && len(s.key) > 0 {
		go func() {
			e <- s.Server.ListenAndServeTLS(s.cert, s.key)
		}()
	} else {
		go func() {
			e <- s.Server.ListenAndServe()
		}()
	}
	if s.feed != nil {
		go s.startTwitter(x)
	}
	go s.Manager.Start(x)
	select {
	case <-w:
	case err := <-e:
		if err != nil {
			s.log.Error("Received error during startup: %s!", err.Error())
		}
	case <-x.Done():
	}
	c()
	s.log.Info("Stopping and shutting down...")
	f, u := context.WithTimeout(x, s.ReadTimeout)
	s.Server.Shutdown(f)
	err := s.Server.Close()
	u()
	close(e)
	close(w)
	return err
}
func (s *Scoreboard) http(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	if len(r.URL.Path) <= 1 || r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := s.html.ExecuteTemplate(w, "home.html", s.Manager.Games); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			s.log.Error("Error during request from %q: %s", r.RemoteAddr, err.Error())
		}
		return
	}
	var (
		v uint64
		n = strings.Trim(r.URL.Path, "/")
		i = strings.IndexRune(n, '/')
	)
	if len(n) == 0 {
		s.fs.ServeHTTP(w, r)
		return
	}
	switch {
	case i < 0:
		v = s.Manager.Game(n)
	case strings.ToLower(n[:i]) == "game":
		if x, err := strconv.Atoi(n[i+1:]); err == nil {
			v = uint64(x)
		}
	}
	if v == 0 {
		s.fs.ServeHTTP(w, r)
		return
	}
	s.log.Debug("Received scoreboard request from %q...", r.RemoteAddr)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.html.ExecuteTemplate(w, "scoreboard.html", &display{Game: v, Twitter: s.feed != nil}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		s.log.Error("Error during request from %q: %s!", r.RemoteAddr, err.Error())
	}
}
func (s *Scoreboard) httpWebsocket(w http.ResponseWriter, r *http.Request) {
	c, err := s.ws.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	s.Manager.New(c)
}
