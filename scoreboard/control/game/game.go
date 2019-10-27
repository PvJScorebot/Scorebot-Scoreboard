package game

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/web"
)

// Game mode constants.
const (
	RedBlue  mode = 0x0
	BlueBlue mode = 0x1
	King     mode = 0x2
	Rush     mode = 0x3
	Defend   mode = 0x4
)

// Game status constants.
const (
	Stopped   status = 0x0
	Running   status = 0x1
	Paused    status = 0x2
	Cancelled status = 0x3
	Completed status = 0x4
)

type mode uint8
type status uint8

// Meta is a struct that represents Game details, such as Name, Start
// and End dates.
type Meta struct {
	ID     int64     `json:"id"`
	End    time.Time `json:"end"`
	Name   string    `json:"name"`
	Mode   mode      `json:"mode"`
	Start  time.Time `json:"start"`
	Status status    `json:"status"`

	hash uint64
}

// Game is a struct that contains all the complex Game data,
// including Hosts and Team information.
type Game struct {
	Meta     *Meta
	Teams    []*team
	Tweets   *tweets
	Events   *events
	Credit   string
	Message  string
	Scorebot string

	hash, total, event uint64
}
type event struct {
	ID   int64             `json:"id"`
	Type uint8             `json:"type"`
	Data map[string]string `json:"data"`
}
type events struct {
	Window  *event
	Current []*event

	hash uint64
}
type tweets struct {
	Tweets []*web.Tweet

	hash uint64
}

// Len helps implement the Sort function.
func (g Game) Len() int {
	return len(g.Teams)
}

// Active is a bool that returns true if the Game is no longer marked as active.
func (m Meta) Active() bool {
	return m.Status != Cancelled && m.Status != Completed
}
func (m mode) String() string {
	switch m {
	case RedBlue:
		return "Red vs Blue"
	case BlueBlue:
		return "Blue vs Blue"
	case King:
		return "King of the Hill"
	case Rush:
		return "Rush"
	case Defend:
		return "Server Defence"
	}
	return "Unknown"
}
func (g *Game) generateHash() {
	sort.Sort(g)
	if g.hash == 0 {
		h := hashers.Get().(*Hasher)
		h.Hash(g.Message)
		g.hash = h.Segment()
		g.Meta.getHash(h)
		for i := range g.Teams {
			if g.Teams[i].Logo == "default.png" {
				g.Teams[i].Logo = "/image/team.png"
			} else {
				g.Teams[i].Logo = fmt.Sprintf("%s%s", g.Scorebot, g.Teams[i].Logo)
			}
			g.Teams[i].getHash(h)
		}
		g.total = h.Sum64()
		h.Reset()
		g.Events.getHash(h)
		h.Reset()
		g.Tweets.getHash(h)
		hashers.Put(h)
	}
}

// Swap helps implement the Sort function.
func (g *Game) Swap(i, j int) {
	g.Teams[i], g.Teams[j] = g.Teams[j], g.Teams[i]
}

// String returns the HTML formatted date/time structs based on the
// null values of this Meta struct.
func (m Meta) String() string {
	if m.Start.IsZero() {
		return ""
	}
	if m.End.IsZero() {
		return fmt.Sprintf(
			"<span>%s</span>",
			m.Start.In(time.UTC).Format("03:04 Jan 2 2006"),
		)
	}
	return fmt.Sprintf(
		"<span>%s</span> to <span>%s</span>",
		m.Start.In(time.UTC).Format("03:04 Jan 2 2006"),
		m.End.In(time.UTC).Format("03:04 Jan 2 2006"),
	)
}
func (s status) String() string {
	switch s {
	case Stopped:
		return "Stopped"
	case Running:
		return "Running"
	case Paused:
		return "Paused"
	case Cancelled:
		return "Cancelled"
	case Completed:
		return "Completed"
	}
	return "Unknown"
}

// Less helps implement the Sort function.
func (g Game) Less(i, j int) bool {
	return g.Teams[i].ID < g.Teams[j].ID
}
func (m *Meta) getHash(h *Hasher) uint64 {
	if m.hash == 0 {
		h.Hash(m.ID)
		h.Hash(m.Mode)
		h.Hash(m.Name)
		h.Hash(m.Status)
		h.Hash(m.End.Unix())
		h.Hash(m.Start.Unix())
		m.hash = h.Segment()
	}
	return m.hash
}
func (e *events) getHash(h *Hasher) uint64 {
	if e.hash == 0 {
		for i := range e.Current {
			h.Hash(e.Current[i].ID)
			h.Hash(e.Current[i].Type)
			for k, v := range e.Current[i].Data {
				h.Hash(k)
				h.Hash(v)
			}
		}
		e.hash = h.Segment()
	}
	return e.hash
}
func (t *tweets) getHash(h *Hasher) uint64 {
	if t.hash == 0 {
		for i := range t.Tweets {
			h.Hash(t.Tweets[i].ID)
		}
		t.hash = h.Segment()
	}
	return t.hash
}

// UnmarshalJSON attempts to unmarshal JSON into a Game struct.
func (g *Game) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	g.Meta = &Meta{}
	g.Tweets = &tweets{}
	g.Events = &events{Current: make([]*event, 0)}
	if x, ok := m["name"]; ok {
		if err := json.Unmarshal(x, &(g.Meta.Name)); err != nil {
			return err
		}
	}
	if x, ok := m["mode"]; ok {
		if err := json.Unmarshal(x, &(g.Meta.Mode)); err != nil {
			return err
		}
	}
	if x, ok := m["credit"]; ok {
		if err := json.Unmarshal(x, &(g.Credit)); err != nil {
			return err
		}
	}
	if x, ok := m["message"]; ok {
		if err := json.Unmarshal(x, &(g.Message)); err != nil {
			return err
		}
	}
	if x, ok := m["teams"]; ok {
		if err := json.Unmarshal(x, &(g.Teams)); err != nil {
			return err
		}
	}
	if x, ok := m["events"]; ok {
		if err := json.Unmarshal(x, &(g.Events.Current)); err != nil {
			return err
		}
	}
	return nil
}
func (g *Game) getDifference(p *planner, old *Game) {
	p.setPrefix("game")
	if old != nil && old.hash == g.hash && len(old.Teams) == len(g.Teams) {
		p.setValue("status", "", "status")
		p.setValue("credit", g.Credit, "game-credit")
		p.setValue("message", g.Message, "game-message")
		g.Meta.getDifference(p, old.Meta)
		g.Events.getDifference(p, old.Events)
		g.Tweets.getDifference(p, old.Tweets)
		for i := range g.Teams {
			g.Teams[i].getDifference(p, old.Teams[i])
		}
	} else {
		p.setDeltaValue("status", "", "status")
		p.setDeltaValue("credit", g.Credit, "game-credit")
		p.setDeltaValue("message", g.Message, "game-message")
		c := make(map[int64]*comparable)
		if old != nil {
			g.Meta.getDifference(p, old.Meta)
			g.Events.getDifference(p, old.Events)
			g.Tweets.getDifference(p, old.Tweets)
			for i := range old.Teams {
				c[old.Teams[i].ID] = &comparable{c1: old.Teams[i]}
			}
		} else {
			g.Meta.getDifference(p, nil)
			g.Events.getDifference(p, nil)
			g.Tweets.getDifference(p, nil)
		}
		for i := range g.Teams {
			v, ok := c[g.Teams[i].ID]
			if !ok {
				v = &comparable{}
				c[g.Teams[i].ID] = v
			}
			v.c2 = g.Teams[i]
		}
		for k, v := range c {
			if v.c2 == nil {
				p.setRemove(fmt.Sprintf("team-t%d", k))
			} else if v.c1 == nil {
				v.c2.(*team).getDifference(p, nil)
			} else {
				v.c2.(*team).getDifference(p, v.c1.(*team))
			}
		}
	}
	p.rollbackPrefix()
}
func (m *Meta) getDifference(p *planner, old *Meta) {
	if old != nil && old.hash == m.hash {
		p.setValue("status-name", m.Name, "game-name")
		p.setValue("status-mode", m.Mode, "game-mode")
		p.setValue("status-status", m.Status, "game-status")
	} else {
		p.setDeltaValue("status-name", m.Name, "game-name")
		p.setDeltaValue("status-mode", m.Mode, "game-mode")
		p.setDeltaValue("status-status", m.Status, "game-status")
	}
}
func (e *events) setWindowEvent(p *planner, w *event) {
	if w.Type <= 0 {
		return
	}
	if e.Window != nil {
		if e.Window.ID == w.ID {
			return
		}
		p.setRemoveEvent(w.ID, w.Type)
	}
	e.Window = w
}
func (e *events) getDifference(p *planner, old *events) {
	if old != nil {
		e.Window = old.Window
	}
	if old != nil && old.hash == e.hash {
		for i := range e.Current {
			p.setEvent(e.Current[i].ID, e.Current[i].Type, e.Current[i].Data)
		}
	} else {
		c := make(map[int64]*comparable)
		if old != nil {
			for i := range old.Current {
				c[old.Current[i].ID] = &comparable{c1: old.Current[i]}
			}
		}
		for i := range e.Current {
			v, ok := c[e.Current[i].ID]
			if !ok {
				v = &comparable{}
				c[e.Current[i].ID] = v
			}
			v.c2 = e.Current[i]
		}
		for k, v := range c {
			if v.c2 == nil {
				p.setRemoveEvent(k, v.c1.(*event).Type)
				continue
			}
			if v.c2.(*event).Type > 0 {
				e.setWindowEvent(p, v.c2.(*event))
			}
			if v.c1 == nil {
				p.setDeltaEvent(k, v.c2.(*event).Type, v.c2.(*event).Data)
			} else {
				p.setEvent(k, v.c2.(*event).Type, v.c2.(*event).Data)
			}
		}
	}
}
func (t *tweets) getDifference(p *planner, old *tweets) {
	if old != nil && old.hash == t.hash {
		for i := range t.Tweets {
			getTweetDifference(p, t.Tweets[i], nil)
		}
	} else {
		c := make(map[int64]*comparable)
		if old != nil {
			for i := range old.Tweets {
				c[old.Tweets[i].ID] = &comparable{c1: old.Tweets[i]}
			}
		}
		for i := range t.Tweets {
			v, ok := c[t.Tweets[i].ID]
			if !ok {
				v = &comparable{}
				c[t.Tweets[i].ID] = v
			}
			v.c2 = t.Tweets[i]
		}
		for k, v := range c {
			if v.c2 == nil {
				p.setRemove(fmt.Sprintf("tweet-t%d", k))
				continue
			}
			if v.c1 == nil {
				getTweetDifference(p, v.c2.(*web.Tweet), nil)
			} else {
				getTweetDifference(p, v.c2.(*web.Tweet), v.c1.(*web.Tweet))
			}
		}
	}
}
func getTweetDifference(p *planner, new, old *web.Tweet) {
	if old == nil {
		p.setDeltaValue(fmt.Sprintf("tweet-t%d", new.ID), "", "tweet")
	} else {
		p.setValue(fmt.Sprintf("tweet-t%d", new.ID), "", "tweet")
	}
	p.setPrefix(fmt.Sprintf("%s-tweet-t%d", p.prefix, new.ID))
	if old != nil {
		p.setValue("pic", "", "tweet-pic")
		p.setProperty("pic-img", fmt.Sprintf("url('%s')", new.UserPhoto), "background-image")
		p.setValue("user", new.User, "tweet-user")
		p.setValue("user-name", new.UserName, "tweet-username")
		p.setValue("user-content", new.Text, "tweet-content")
		p.setValue("image", "", "tweet-media")
		for x := range new.Images {
			p.setValue(fmt.Sprintf("image-%d", x), "", "tweet-image")
			p.setProperty(fmt.Sprintf("image-%d", x), fmt.Sprintf("url('%s')", new.Images[x]), "background-image")
		}
	} else {
		p.setDeltaValue("pic", "", "tweet-pic")
		p.setDeltaProperty("pic-img", fmt.Sprintf("url('%s')", new.UserPhoto), "background-image")
		p.setDeltaValue("user", new.User, "tweet-user")
		p.setDeltaValue("user-name", new.UserName, "tweet-username")
		p.setDeltaValue("user-content", new.Text, "tweet-content")
		p.setDeltaValue("image", "", "tweet-media")
		for x := range new.Images {
			p.setDeltaValue(fmt.Sprintf("image-%d", x), "", "tweet-image")
			p.setDeltaProperty(fmt.Sprintf("image-%d", x), fmt.Sprintf("url('%s')", new.Images[x]), "background-image")
		}
	}
	p.rollbackPrefix()
}

// Difference returns two sets of Update arrays, the first is the required updates to build
// the current board (for new clients) and the second is the delta updates that need to be sent to existing clients.
func (g *Game) Difference(old *Game) ([]*Update, []*Update) {
	p := &planner{
		Delta:  make([]*Update, 0),
		Create: make([]*Update, 0),
	}
	g.generateHash()
	g.getDifference(p, old)
	return p.Create, p.Delta
}
