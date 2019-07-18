package game

import (
	"encoding/json"
	"fmt"
	"time"
)

// Game mode constants.
const (
	RedBlue  Mode = 0x0
	BlueBlue Mode = 0x1
	King     Mode = 0x2
	Rush     Mode = 0x3
	Defend   Mode = 0x4
)

// Game status constants.
const (
	Stopped   Status = 0x0
	Running   Status = 0x1
	Paused    Status = 0x2
	Cancled   Status = 0x3
	Completed Status = 0x4
)

// Mode is an integer repersentation of the Game mode type.
type Mode uint8

// Status is an integer repersentaton of the Game running status.
type Status uint8

// Meta is a struct that repersents Game details, such as Name, Start
// and End dates.
type Meta struct {
	ID     int64     `json:"id"`
	End    time.Time `json:"end"`
	Name   string    `json:"name"`
	Mode   Mode      `json:"mode"`
	Start  time.Time `json:"start"`
	Status Status    `json:"status"`

	hash uint32
}

// Game is a struct that contains all the complex Game data,
// including Hosts and Team information.
type Game struct {
	Meta    *Meta
	Teams   []*Team
	Credit  string
	Message string

	hash  uint32
	total uint32
}

// String returns the proper name for this Mode.
func (m Mode) String() string {
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

// GenerateHash returns the total game hash and generates the individal hash value for each
// sub item.
func (g *Game) GenerateHash() {
	if g.hash == 0 {
		h := &Hasher{}
		h.Hash(g.Message)
		g.hash = h.Segment()
		g.Meta.getHash(h)
		for i := range g.Teams {
			g.Teams[i].getHash(h)
		}
		g.total = h.Sum32()
	}
}

// String returns the HTML formatted date/time structs based on the
// null values of this Meta struct.
func (m *Meta) String() string {
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

// String returns the proper name for this Status type.
func (s Status) String() string {
	switch s {
	case Stopped:
		return "Stopped"
	case Running:
		return "Running"
	case Paused:
		return "Paused"
	case Cancled:
		return "Cancled"
	case Completed:
		return "Completed"
	}
	return "Unknown"
}
func (m *Meta) getHash(h *Hasher) uint32 {
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

// UnmarshalJSON attempts to unmarshal JSON into a Game struct.
func (g *Game) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	g.Meta = &Meta{}
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
	return nil
}
func (g *Game) getDifference(p *planner, old *Game) {
	p.setPrefix("game")
	if old != nil && old.hash == g.hash {
		p.setValue("status", "", "status")
		p.setValue("credit", g.Credit, "game-credit")
		p.setValue("message", g.Message, "game-message")
		g.Meta.getDifference(p, old.Meta)
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
			for i := range old.Teams {
				c[old.Teams[i].ID] = &comparable{c1: old.Teams[i]}
			}
		} else {
			g.Meta.getDifference(p, nil)
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
				v.c2.(*Team).getDifference(p, nil)
			} else {
				v.c2.(*Team).getDifference(p, v.c1.(*Team))
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

// Difference returns two sets of Update arrays, the first is the required updates to build
// the current board (for new clients) and the second is the delta updates that need to be sent to existing clients.
func (g *Game) Difference(old *Game) (new []*Update, delta []*Update) {
	p := &planner{
		Delta:  make([]*Update, 0),
		Create: make([]*Update, 0),
	}
	g.getDifference(p, old)
	return p.Create, p.Delta
}