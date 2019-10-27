package scoreboard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/iDigitalFlame/logx/logx"
	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/control"
	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/control/game"
	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/web"

	"github.com/gobuffalo/packr/v2"
	"github.com/stvp/slug"
)

const (
	// Seperator is a comma constant, used to split keyword parameters.
	Seperator = ","

	webSockBufferSize = 2048
)

var (
	// ErrInvalidTick is returned if the specified tick value is less than or equal to zero.
	ErrInvalidTick = errors.New("tick rate must be grater than zero")
	// ErrInvalidConfig is returned if a passed Config struct is nil.
	ErrInvalidConfig = errors.New("config struct cannot be nil")
	// ErrInvalidLevel is returned if the specified log level value is not in the bounds of [0 - 5].
	ErrInvalidLevel = errors.New("level must be between 0 and 5 inclusive")

	resources = packr.New("html", "../html")
)

type display struct {
	Game    int64
	Twitter bool
}

// Scoreboard is a struct that represents the Scoreboard multiplexer.
// This struct is used to gather and compare Game data to push to Scoreboard
// clients.
type Scoreboard struct {
	ctx        context.Context
	api        *web.API
	err        error
	log        logx.Log
	html       *template.Template
	tick       time.Duration
	games      []*game.Meta
	names      map[string]int64
	timer      *time.Timer
	assets     string
	signal     chan os.Signal
	cancel     context.CancelFunc
	server     *web.Server
	twitter    *web.Twitter
	timeout    time.Duration
	collection *control.Collection
}

func init() {
	slug.Replacement = '-'
}

// Start begins the listening process for the Scoreboard.  This function
// blocks until interrupted.
func (s *Scoreboard) Start() error {
	s.log.Info("Starting Scoreboard service...")
	go func(i *Scoreboard) {
		defer func() { recover() }()
		if err := i.server.Start(); err != nil && err != http.ErrServerClosed {
			i.log.Error("Web server returned error: %s", err.Error())
			if i.ctx.Err() == nil {
				i.err = err
				i.signal <- syscall.SIGTERM
			}
		}
	}(s)
	select {
	case <-s.signal:
	case <-s.ctx.Done():
	}
	s.log.Info("Stopping and shutting down...")
	s.cancel()
	s.timer.Stop()
	s.server.Stop()
	close(s.signal)
	if s.twitter != nil {
		s.twitter.Stop()
	}
	s.collection.Stop()
	return s.err
}
func (s *Scoreboard) update() error {
	defer func(l logx.Log) {
		if err := recover(); err != nil {
			l.Error("Update function recovered from a panic: %s", err)
		}
	}(s.log)
	s.log.Trace("Starting update...")
	if err := s.api.GetJSON("api/games/", &(s.games)); err != nil {
		s.log.Error("Error occurred during tick: %s", err.Error())
		return err
	}
	for i := range s.games {
		n := slug.Clean(s.games[i].Name)
		if !s.games[i].Active() {
			if _, ok := s.names[n]; ok {
				delete(s.names, n)
			}
			continue
		}
		if _, ok := s.names[n]; !ok {
			s.names[n] = s.games[i].ID
			s.log.Debug("Added game name mapping \"%s\" to ID %d.", n, s.games[i].ID)
		}
	}
	s.log.Debug("Read %d games from scorebot, update finished.", len(s.games))
	s.collection.Sync(s.timeout)
	return nil
}

// New creates a new Scoreboard instance from the supplied
// Config struct.
func New(c *Config) (*Scoreboard, error) {
	if c == nil {
		return nil, ErrInvalidConfig
	}
	if err := c.verify(); err != nil {
		return nil, err
	}
	x := time.Second * time.Duration(c.Timeout)
	a, err := web.NewAPI(c.Scorebot, x, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to setup API: %w", err)
	}
	var p, t string
	if len(c.Directory) > 0 {
		p = filepath.Join(c.Directory, "public")
		if d, err := os.Stat(p); err != nil || !d.IsDir() {
			return nil, fmt.Errorf("public directory \"%s\" is not valid", p)
		}
		t = filepath.Join(c.Directory, "template")
	}
	z := template.New("base")
	if err := getTemplate(z, t, "home.html"); err != nil {
		return nil, err
	}
	if err := getTemplate(z, t, "scoreboard.html"); err != nil {
		return nil, err
	}
	s := &Scoreboard{
		api:     a,
		tick:    time.Duration(time.Second * time.Duration(c.Tick)),
		html:    z,
		names:   make(map[string]int64),
		assets:  c.Assets,
		signal:  make(chan os.Signal),
		timeout: x,
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	signal.Notify(s.signal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	if s.server, err = web.NewServer(s.ctx, x, c.Listen, p, c.Cert, c.Key, resources); err != nil {
		return nil, fmt.Errorf("unable to setup web server: %w", err)
	}
	if len(c.Log.File) > 0 {
		f, err := logx.NewFile(logx.Level(c.Log.Level), c.Log.File)
		if err != nil {
			return nil, fmt.Errorf("unable to create log file \"%s\": %w", c.Log.File, err)
		}
		s.log = logx.NewStack(f, logx.NewConsole(logx.Level(c.Log.Level)))
	} else {
		s.log = logx.NewConsole(logx.Level(c.Log.Level))
	}
	s.collection = control.NewCollection(s.ctx, s.api, s.log)
	s.collection.Callback = s.updateMeta
	s.server.AddHandlerFunc("/", s.http)
	s.server.AddHandler("/w", web.NewWebSocket(webSockBufferSize, s.collection.NewClient))
	if err := s.update(); err != nil {
		s.log.Warning("Initial connection was unable to connect to scorebot \"%s\": %s", c.Scorebot, err.Error())
	}
	if c.Twitter != nil {
		x, err := web.NewTwitter(s.ctx, time.Duration(c.Twitter.Timeout)*time.Second, c.Twitter.Filter, c.Twitter.Credentials)
		if err != nil {
			return nil, fmt.Errorf("unable to setup Twitter: %w", err)
		}
		s.twitter = x
		s.twitter.Callback = s.collection.SetupTwitter(time.Duration(c.Twitter.Expire) * time.Second)
		if err := s.twitter.Start(); err != nil {
			return nil, fmt.Errorf("unable to start Twitter stream: %w", err)
		}
		s.log.Info("Twitter setup complete!")
	} else {
		s.log.Info("Missing Twitter Keys and Filter Parameters, skipping Twitter setup!")
	}
	s.timer = time.AfterFunc(s.tick, func() {
		s.update()
		if s.ctx.Err() == nil {
			s.timer.Reset(s.tick)
		}
	})
	return s, nil
}
func (s Scoreboard) updateMeta(g *game.Game) {
	if len(s.assets) > 0 {
		g.Scorebot = s.assets
	} else {
		g.Scorebot = s.api.String()
	}
	for i := range s.games {
		if s.games[i].ID == g.Meta.ID {
			g.Meta.End = s.games[i].End
			g.Meta.Start = s.games[i].Start
			g.Meta.Status = s.games[i].Status
			return
		}
	}
}
func getTemplate(t *template.Template, d, f string) error {
	if len(d) > 0 {
		s := filepath.Join(d, f)
		if i, err := os.Stat(s); err == nil && !i.IsDir() {
			_, err := t.New(f).ParseFiles(s)
			if err != nil {
				return fmt.Errorf("unable to parse templates \"%s\": %w", f, err)
			}
			return nil
		}
	}
	c, err := resources.FindString(fmt.Sprintf("template/%s", f))
	if err != nil {
		return fmt.Errorf("could not find template \"%s\": %w", f, err)
	}
	if _, err := t.New(f).Parse(c); err != nil {
		return fmt.Errorf("unable to parse scorebot template \"%s\": %w", f, err)
	}
	return nil
}
func (s *Scoreboard) http(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, http.StatusText(http.StatusMethodNotAllowed))
		return
	}
	if r.URL.Path == "/" {
		if err := s.html.ExecuteTemplate(w, "home.html", s.games); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, http.StatusText(http.StatusInternalServerError))
			s.log.Error("Error during request from \"%s\": %s", r.RemoteAddr, err.Error())
		}
		return
	}
	var z int64
	n := strings.Trim(r.URL.Path, "/")
	i := strings.IndexRune(n, '/')
	if i < 0 {
		if g, ok := s.names[slug.Clean(n)]; ok {
			z = g
		}
	} else if strings.ToLower(n[:i]) == "game" {
		if i, err := strconv.ParseInt(n[i+1:], 10, 64); err == nil {
			z = i
		}
	}
	if z > 0 {
		s.log.Debug("Received scoreboard request from \"%s\"...", r.RemoteAddr)
		if err := s.html.ExecuteTemplate(w, "scoreboard.html", &display{Game: z, Twitter: s.twitter != nil}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, http.StatusText(http.StatusInternalServerError))
			s.log.Error("Error during request from \"%s\": %s", r.RemoteAddr, err.Error())
		}
		return
	}
	s.server.ServeHTTP(w, r)
}
