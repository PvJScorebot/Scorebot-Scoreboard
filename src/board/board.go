package board

import (
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

	"./control"
	"./control/game"
	"./logging"
	"./web"
	"golang.org/x/xerrors"

	"github.com/stvp/slug"
)

var (
	// ErrInvalidTick is returned by 'NewScorebot' if the specified tick value is less than or equal to zero.
	ErrInvalidTick = errors.New("tick rate must be grater than zero")
)

// Scoreboard is a struct that repersents the Scoreboard multiplexer.
// This struct is used to gather and compare Game data to push to Scoreboard
// clients.
type Scoreboard struct {
	API        *web.API
	Server     *web.Server
	Collection *control.Collection

	log   logging.Log
	html  *template.Template
	tick  time.Duration
	games []*game.Meta
	names map[string]int64
	timer *time.Timer
}

func init() {
	slug.Replacement = '-'
}

// Start begins the listening process for the Scoreboard.  This function
// blocks untill interrupted.
func (s *Scoreboard) Start() error {
	s.log.Info("Starting scoreboard service..")
	go s.Server.Start()
	wait := make(chan os.Signal)
	signal.Notify(wait, syscall.SIGINT, syscall.SIGTERM)
	<-wait
	s.log.Info("Stopping and shutting down..")
	s.timer.Stop()
	return s.Collection.Done()
}
func (s *Scoreboard) update() error {
	s.log.Debug("Starting update..")
	if err := s.API.GetJSON("api/games/", &(s.games)); err != nil {
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
	s.Collection.Sync()
	return nil
}
func (s *Scoreboard) updateMeta(g *game.Game) {
	g.Scorebot = s.API.Base.String()
	for i := range s.games {
		if s.games[i].ID == g.Meta.ID {
			g.Meta.End = s.games[i].End
			g.Meta.Start = s.games[i].Start
			g.Meta.Status = s.games[i].Status
			return
		}
	}
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
		if err := s.html.ExecuteTemplate(w, "scoreboard.html", z); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, http.StatusText(http.StatusInternalServerError))
			s.log.Error("Error during request from \"%s\": %s", r.RemoteAddr, err.Error())
		}
		return
	}
	s.Server.ServeHTTP(w, r)
}

// NewScoreboard creates a Scoreboard instance from the asssets
// directory 'd' and the Scorebot address 's'. Errors are non-nil if the
// specified directory does not exist or lacks permissions.
func NewScoreboard(listen string, timeout, tick int, dir, scorebot, log string, level int) (*Scoreboard, error) {
	if tick <= 0 {
		return nil, ErrInvalidTick
	}
	p, t := filepath.Join(dir, "public"), filepath.Join(dir, "template")
	if d, err := os.Stat(p); err != nil || !d.IsDir() {
		return nil, xerrors.Errorf("public directory \"%s\" is not a valid directory", p)
	}
	if d, err := os.Stat(t); err != nil || !d.IsDir() {
		return nil, xerrors.Errorf("templates directory \"%s\" is not a valid directory", t)
	}
	a, err := web.NewAPI(scorebot, time.Second*time.Duration(timeout), nil)
	if err != nil {
		return nil, xerrors.Errorf("unable to setup API: %w", err)
	}
	w, err := web.NewServer(listen, p)
	if err != nil {
		return nil, xerrors.Errorf("unable to setup web server: %w", err)
	}
	z, err := template.ParseFiles(
		filepath.Join(t, "home.html"),
		filepath.Join(t, "scoreboard.html"),
	)
	if err != nil {
		return nil, xerrors.Errorf("unable to parse scorebot templates: %w", err)
	}
	s := &Scoreboard{
		API:    a,
		tick:   time.Duration(time.Second * time.Duration(tick)),
		html:   z,
		names:  make(map[string]int64),
		Server: w,
	}
	if len(log) > 0 {
		f, err := logging.NewFile(logging.Level(level), log)
		if err != nil {
			return nil, fmt.Errorf("unable to create log file \"%s\": %s", log, err.Error())
		}
		s.log = logging.NewStack(f, logging.NewConsole(logging.Level(level)))

	} else {
		s.log = logging.NewConsole(logging.Level(level))
	}
	s.Collection = control.NewCollection(s.API, s.log)
	s.Collection.GameCallback(s.updateMeta)
	s.Server.AddHandlerFunc("/", s.http)
	s.Server.AddHandler("/w", web.NewWebSocket(2048, func(u *web.Stream) { s.Collection.NewClient(u) }))
	if err := s.update(); err != nil {
		return nil, xerrors.Errorf("unable to connect to scorebot \"%s\": %w", scorebot, err)
	}
	s.timer = time.AfterFunc(s.tick, func() {
		s.update()
		s.timer.Reset(s.tick)
	})
	return s, nil
}
