package scoreboard

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/control"
	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/control/game"
	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/logging"
	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/web"

	"github.com/gobuffalo/packr/v2"
	"github.com/stvp/slug"

	"golang.org/x/xerrors"
)

const (
	// ConfigSeperator is a comma constant, used to split keyword parameters.
	ConfigSeperator = ","

	// DefaultTick is the default tick time in seconds. Used if the tick setting is missing.
	DefaultTick uint16 = 5
	// DefaultExpire is the default tweet timeout. Used if the Twitter.expire setting is missing.
	DefaultExpire uint16 = 45
	// DefaultListen is the default listen address. Used if the listen setting is missing.
	DefaultListen string = "0.0.0.0:8080"
	// DefaultTimeout is the default timeout in seconds. Used if the timeout setting is missing.
	DefaultTimeout uint16 = 10
	// DefaultLogLevel is the default log level. Used if the log.level setting is missing.
	DefaultLogLevel uint8 = 2
)

var (
	// ErrInvalidTick is returned if the specified tick value is less than or equal to zero.
	ErrInvalidTick = xerrors.New("tick rate must be grater than zero")
	// ErrInvalidConfig is returned if a passed Config struct is nil.
	ErrInvalidConfig = xerrors.New("config struct cannot be nil")
	// ErrInvalidLevel is returned if the specified log level value is not in the bounds of [0 - 5].
	ErrInvalidLevel = xerrors.New("level must be between 0 and 5 inclusive")

	resources = packr.New("html", "../html")
)

// Log is a struct that stores and repersents the Scoreboard Logging config
// Able to be loaded from JSON
type Log struct {
	File  string `json:"file"`
	Level uint8  `json:"level"`
}

// Config is a struct that stores and repersents the Scoreboard config
// Able to be loaded from JSON.
type Config struct {
	Log       *Log     `json:"log,omitempty"`
	Tick      uint16   `json:"tick"`
	Assets    string   `json:"assets"`
	Listen    string   `json:"listen"`
	Twitter   *Twitter `json:"twitter,omitempty"`
	Timeout   uint16   `json:"timeout"`
	KeyFile   string   `json:"key"`
	Scorebot  string   `json:"scorebot"`
	CertFile  string   `json:"cert"`
	Directory string   `json:"dir"`
}
type display struct {
	Game    int64
	Twitter bool
}

// Twitter is a struct that stores and repersents the Scoreboard Twitter config
// Able to be loaded from JSON.
type Twitter struct {
	Filter      *web.Filter      `json:"filter"`
	Expire      uint16           `json:"expire"`
	Timeout     uint16           `json:"timeout"`
	Credentials *web.Credentials `json:"auth"`
}

// Scoreboard is a struct that repersents the Scoreboard multiplexer.
// This struct is used to gather and compare Game data to push to Scoreboard
// clients.
type Scoreboard struct {
	api        *web.API
	log        logging.Log
	html       *template.Template
	tick       time.Duration
	games      []*game.Meta
	names      map[string]int64
	timer      *time.Timer
	assets     string
	server     *web.Server
	twitter    *web.Twitter
	timeout    time.Duration
	collection *control.Collection
}

func init() {
	slug.Replacement = '-'
}

// Defaults returns a JSON string repersentation of the default config.
// Used for creating and understanding the config file structure.
func Defaults() string {
	c := &Config{
		Log: &Log{
			File:  "",
			Level: DefaultLogLevel,
		},
		Tick:   DefaultTick,
		Assets: "",
		Listen: DefaultListen,
		Twitter: &Twitter{
			Filter: &web.Filter{
				Language: []string{"en"},
				Keywords: []string{
					"pvj",
					"ctf",
				},
				OnlyUsers:    []string{},
				BlockedUsers: []string{},
				BlockedWords: []string{},
			},
			Expire:  DefaultExpire,
			Timeout: DefaultTimeout,
			Credentials: &web.Credentials{
				AccessKey:      "",
				ConsumerKey:    "",
				AccessSecret:   "",
				ConsumerSecret: "",
			},
		},
		Timeout:   DefaultTimeout,
		KeyFile:   "",
		Scorebot:  "http://scorebot",
		CertFile:  "",
		Directory: "html",
	}
	b, _ := json.MarshalIndent(c, "", "    ")
	return string(b)
}
func (c *Config) verify() error {
	if c.Tick <= 0 {
		return ErrInvalidTick
	}
	if c.Timeout < 0 {
		return web.ErrInvalidTimeout
	}
	if c.Log != nil {
		if c.Log.Level < 0 || c.Log.Level > 5 {
			return ErrInvalidLevel
		}
	} else {
		c.Log = &Log{Level: DefaultLogLevel}
	}
	if c.Twitter != nil {
		v := true
		if c.Twitter.Credentials != nil {
			if len(c.Twitter.Credentials.AccessKey) == 0 || len(c.Twitter.Credentials.AccessSecret) == 0 {
				v = false
			}
			if len(c.Twitter.Credentials.ConsumerKey) == 0 || len(c.Twitter.Credentials.ConsumerSecret) == 0 {
				v = false
			}
		} else {
			v = false
		}
		if c.Twitter.Filter != nil {
			if len(c.Twitter.Filter.Language) == 0 || len(c.Twitter.Filter.Keywords) == 0 {
				v = false
			}
		} else {
			v = false
		}
		if !v {
			c.Twitter = nil
		}
	}
	return nil
}

// Start begins the listening process for the Scoreboard.  This function
// blocks untill interrupted.
func (s *Scoreboard) Start() error {
	s.log.Info("Starting scoreboard service..")
	w := make(chan os.Signal)
	signal.Notify(w, syscall.SIGINT, syscall.SIGTERM)
	go func(z *Scoreboard, q chan os.Signal) {
		if err := s.server.Start(); err != nil {
			z.log.Error("Web server returned error: %s", err.Error())
			w <- syscall.SIGTERM
		}
	}(s, w)
	<-w
	s.log.Info("Stopping and shutting down..")
	s.timer.Stop()
	if s.twitter != nil {
		s.twitter.Stop()
	}
	return s.collection.Stop()
}
func (s *Scoreboard) update() error {
	defer func(l logging.Log) {
		if err := recover(); err != nil {
			l.Error("update gofunc: recovered from a panic: %s", err)
		}
	}(s.log)
	s.log.Debug("Starting update..")
	if err := s.api.GetJSON("api/games/", &(s.games)); err != nil {
		s.log.Error("Error occured during tick: %s", err.Error())
		return err
	}
	for i := range s.games {
		if !s.games[i].Active() {
			continue
		}
		n := slug.Clean(s.games[i].Name)
		if _, ok := s.names[n]; !ok {
			s.names[n] = s.games[i].ID
			s.log.Debug("Added game name mapping \"%s\" to ID %d.", n, s.games[i].ID)
		}
	}
	s.log.Debug("Read %d games from scorebot, update finished.", len(s.games))
	s.collection.Sync(s.timeout)
	return nil
}

// Load loads the config from the specified file path 's'
func Load(s string) (*Config, error) {
	f, err := os.Stat(s)
	if err != nil {
		return nil, xerrors.Errorf("cannot load file \"%s\": %w", s, err)
	}
	if f.IsDir() {
		return nil, xerrors.Errorf("cannot load \"%s\": path is not a file", s)
	}
	b, err := ioutil.ReadFile(s)
	if err != nil {
		return nil, xerrors.Errorf("cannot read file \"%s\": %w", s, err)
	}
	var c *Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, xerrors.Errorf("cannot read file \"%s\" into JSON: %w", s, err)
	}
	return c, nil
}

// SplitParm returns a string array from a comma seperated list.
// This function also trims the string lengths of excess spaces.
func SplitParm(s, d string) []string {
	if len(s) == 0 {
		return []string{}
	}
	f := strings.Split(s, d)
	for i := range f {
		f[i] = strings.TrimSpace(f[i])
	}
	return f
}
func (s *Scoreboard) updateMeta(g *game.Game) {
	if len(s.assets) > 0 {
		g.Scorebot = s.assets
	} else {
		g.Scorebot = s.api.Base.String()
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

// NewScoreboard creates a Scoreboard instance from the asssets
// directory 'd' and the Scorebot address 's'. Errors are non-nil if the
// specified directory does not exist or lacks permissions.
func NewScoreboard(c *Config) (*Scoreboard, error) {
	if c == nil {
		return nil, ErrInvalidConfig
	}
	if err := c.verify(); err != nil {
		return nil, err
	}
	if len(c.Listen) == 0 {
		c.Listen = DefaultListen
	}
	x := time.Second * time.Duration(c.Timeout)
	a, err := web.NewAPI(c.Scorebot, x, nil)
	if err != nil {
		return nil, xerrors.Errorf("unable to setup API: %w", err)
	}
	p := ""
	z := template.New("base")
	if len(c.Directory) > 0 {
		p = filepath.Join(c.Directory, "public")
		if d, err := os.Stat(p); err != nil || !d.IsDir() {
			return nil, xerrors.Errorf("public directory \"%s\" is not a valid directory", p)
		}
		t := filepath.Join(c.Directory, "template")
		getTemplate(z, "home.html", t, "home.html")
		getTemplate(z, "scoreboard.html", t, "scoreboard.html")
	} else {
		getTemplate(z, "home.html", "", "home.html")
		getTemplate(z, "scoreboard.html", "", "scoreboard.html")
	}
	w, err := web.NewServer(c.Listen, p, c.CertFile, c.KeyFile, resources)
	if err != nil {
		return nil, xerrors.Errorf("unable to setup web server: %w", err)
	}
	s := &Scoreboard{
		api:     a,
		tick:    time.Duration(time.Second * time.Duration(c.Tick)),
		html:    z,
		names:   make(map[string]int64),
		assets:  c.Assets,
		server:  w,
		timeout: x,
	}
	if c.Log != nil {
		if len(c.Log.File) > 0 {
			f, err := logging.NewFile(logging.Level(c.Log.Level), c.Log.File)
			if err != nil {
				return nil, fmt.Errorf("unable to create log file \"%s\": %s", c.Log.File, err.Error())
			}
			s.log = logging.NewStack(f, logging.NewConsole(logging.Level(c.Log.Level)))
		} else {
			s.log = logging.NewConsole(logging.Level(c.Log.Level))
		}
	} else {
		s.log = logging.NewConsole(logging.Level(DefaultLogLevel))
	}
	s.collection = control.NewCollection(s.api, s.log)
	s.collection.GameCallback(s.updateMeta)
	s.server.AddHandlerFunc("/", s.http)
	s.server.AddHandler("/w", web.NewWebSocket(2048, s.collection.NewClient))
	if err := s.update(); err != nil {
		return nil, xerrors.Errorf("unable to connect to scorebot \"%s\": %w", c.Scorebot, err)
	}
	if c.Twitter != nil {
		x, err := web.NewTwitter(time.Duration(c.Twitter.Timeout)*time.Second, c.Twitter.Filter, c.Twitter.Credentials)
		if err != nil {
			return nil, xerrors.Errorf("unable to setup Twitter: %w", err)
		}
		s.twitter = x
		s.twitter.Callback(s.collection.SetupTwitter(time.Duration(c.Twitter.Expire) * time.Second))
		if err := s.twitter.Start(); err != nil {
			return nil, xerrors.Errorf("unable to start Twitter stream: %w", err)
		}
		s.log.Info("Twitter setup complete!")
	} else {
		s.log.Info("Missing Twitter Keys and Filter Parameters, skipping Twitter setup!")
	}
	s.timer = time.AfterFunc(s.tick, func() {
		s.update()
		s.timer.Reset(s.tick)
	})
	return s, nil
}
func getTemplate(t *template.Template, n, d, f string) error {
	if len(d) > 0 {
		s := filepath.Join(d, f)
		i, err := os.Stat(s)
		if err == nil && !i.IsDir() {
			_, err := t.New(n).ParseFiles(s)
			return xerrors.Errorf("unable to parse templates \"%s\": %w", f, err)
		}
	}
	c, err := resources.FindString(fmt.Sprintf("template/%s", f))
	if err != nil {
		return xerrors.Errorf("could not find template \"%s\": %w", f, err)
	}
	if _, err := t.New(n).Parse(c); err != nil {
		return xerrors.Errorf("unable to parse scorebot templates: %w", err)
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
		s.log.Debug("Received scoreboard request from \"%s\"..", r.RemoteAddr)
		if err := s.html.ExecuteTemplate(w, "scoreboard.html", &display{Game: z, Twitter: s.twitter != nil}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, http.StatusText(http.StatusInternalServerError))
			s.log.Error("Error during request from \"%s\": %s", r.RemoteAddr, err.Error())
		}
		return
	}
	s.server.ServeHTTP(w, r)
}
