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
	"encoding/json"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	redBlue  mode = 0x0
	blueBlue mode = 0x1
	king     mode = 0x2
	rush     mode = 0x3
	defend   mode = 0x4

	stopped   status = 0x0
	running   status = 0x1
	paused    status = 0x2
	cancelled status = 0x3
	completed status = 0x4
)

var (
	hashers = sync.Pool{
		New: func() interface{} {
			return new(hasher)
		},
	}

	emptyMeta meta
)

type mode uint8
type status uint8
type meta struct {
	End   time.Time `json:"end"`
	Start time.Time `json:"start"`
	Name  string    `json:"name"`

	ID   uint64 `json:"id"`
	hash uint64

	Mode   mode   `json:"mode"`
	Status status `json:"status"`
}
type game struct {
	Meta    meta
	Teams   []team
	Tweets  []tweet
	Events  events
	Credit  string
	Message string

	hash, total, tweets uint64
}

func (g game) Len() int {
	return len(g.Teams)
}
func (m meta) Active() bool {
	return m.Status != cancelled && m.Status != completed
}
func (m mode) String() string {
	switch m {
	case redBlue:
		return "Red vs Blue"
	case blueBlue:
		return "Blue vs Blue"
	case king:
		return "King of the Hill"
	case rush:
		return "Rush"
	case defend:
		return "Server Defence"
	}
	return "Unknown"
}
func (g *game) Swap(i, j int) {
	g.Teams[i], g.Teams[j] = g.Teams[j], g.Teams[i]
}
func (m meta) String() string {
	if m.Start.IsZero() {
		return ""
	}
	if m.End.IsZero() {
		return "<span>" + m.Start.In(time.UTC).Format("03:04 Jan 2 2006") + "</span>"
	}
	return "<span>" + m.Start.In(time.UTC).Format("03:04 Jan 2 2006") + "</span> to <span>" +
		m.End.In(time.UTC).Format("03:04 Jan 2 2006") + "</span>"
}
func (s status) String() string {
	switch s {
	case stopped:
		return "Stopped"
	case running:
		return "Running"
	case paused:
		return "Paused"
	case cancelled:
		return "Cancelled"
	case completed:
		return "Completed"
	}
	return "Unknown"
}
func (g game) Less(i, j int) bool {
	return g.Teams[i].ID < g.Teams[j].ID
}
func (m *meta) Hash(h *hasher) uint64 {
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
func (g *game) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	if x, ok := m["name"]; ok {
		if err := json.Unmarshal(x, &g.Meta.Name); err != nil {
			return err
		}
	}
	if x, ok := m["mode"]; ok {
		if err := json.Unmarshal(x, &g.Meta.Mode); err != nil {
			return err
		}
	}
	if x, ok := m["credit"]; ok {
		if err := json.Unmarshal(x, &g.Credit); err != nil {
			return err
		}
	}
	if x, ok := m["message"]; ok {
		if err := json.Unmarshal(x, &g.Message); err != nil {
			return err
		}
	}
	if x, ok := m["teams"]; ok {
		if err := json.Unmarshal(x, &g.Teams); err != nil {
			return err
		}
	}
	if x, ok := m["events"]; ok {
		if err := json.Unmarshal(x, &g.Events.Current); err != nil {
			return err
		}
	}
	return nil
}
func (g *game) Compare(p *planner, o *game) {
	p.Prefix("game")
	if o != nil && o.hash == g.hash && len(o.Teams) == len(g.Teams) {
		p.Value("status", "", "status")
		p.Value("credit", g.Credit, "game-credit")
		p.Value("message", g.Message, "game-message")
		g.Meta.Compare(p, o.Meta)
		g.Events.Compare(p, o.Events)
		g.compareTweets(p, o)
		for i := range g.Teams {
			g.Teams[i].Compare(p, o.Teams[i])
		}
		p.rollbackPrefix()
		return
	}
	p.DeltaValue("status", "", "status")
	p.DeltaValue("credit", g.Credit, "game-credit")
	p.DeltaValue("message", g.Message, "game-message")
	c := make(compare)
	if o != nil {
		g.Meta.Compare(p, o.Meta)
		g.Events.Compare(p, o.Events)
		g.compareTweets(p, o)
		for i := range o.Teams {
			c.One(o.Teams[i])
		}
	} else {
		g.Meta.Compare(p, emptyMeta)
		g.Events.Compare(p, emptyEvents)
		g.compareTweets(p, nil)
	}
	for i := range g.Teams {
		c.Two(g.Teams[i])
	}
	for k, v := range c {
		switch {
		case !v.Second():
			p.Remove("team-t" + strconv.FormatUint(k, 64))
		case !v.First():
			v.B.(team).Compare(p, emptyTeam)
		default:
			v.B.(team).Compare(p, v.A.(team))
		}
	}
	p.rollbackPrefix()
}
func (m meta) Compare(p *planner, old meta) {
	if old.ID != 0 && old.hash == m.hash {
		p.Value("status-name", m.Name, "game-name")
		p.Value("status-mode", m.Mode, "game-mode")
		p.Value("status-status", m.Status, "game-status")
		return
	}
	p.DeltaValue("status-name", m.Name, "game-name")
	p.DeltaValue("status-mode", m.Mode, "game-mode")
	p.DeltaValue("status-status", m.Status, "game-status")
}
func (g *game) Delta(s string, old *game) ([]update, []update) {
	p := new(planner)
	sort.Sort(g)
	if g.hash == 0 {
		h := hashers.Get().(*hasher)
		h.Hash(g.Message)
		g.hash = h.Segment()
		g.Meta.Hash(h)
		for i := range g.Teams {
			if g.Teams[i].Logo == "default.png" || len(g.Teams[i].Logo) == 0 {
				g.Teams[i].Logo = "/image/team.png"
			} else {
				g.Teams[i].Logo = s + g.Teams[i].Logo
			}
			g.Teams[i].Hash(h)
		}
		g.total = h.Sum64()
		h.Reset()
		g.Events.Hash(h)
		h.Reset()
		g.hashTweets(h)
		hashers.Put(h)
	}
	g.Compare(p, old)
	return p.Create, p.Delta
}
