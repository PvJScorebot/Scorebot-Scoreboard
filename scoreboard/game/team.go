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
	"strconv"
)

var (
	emptyTeam   team
	emptyBeacon beacon
)

type team struct {
	Name    string      `json:"name"`
	Logo    string      `json:"logo"`
	Color   string      `json:"color"`
	Beacons []beacon    `json:"beacons"`
	Hosts   []host      `json:"hosts"`
	Flags   scoreFlag   `json:"flags"`
	Score   score       `json:"score"`
	Tickets scoreTicket `json:"tickets"`
	ID      uint64      `json:"id"`
	hash    uint64
	total   uint64
	Minimal bool `json:"minimal"`
	Offense bool `json:"offense"`
}
type beacon struct {
	Color string `json:"color"`
	ID    uint64 `json:"id"`
	Team  uint64 `json:"team"`
	hash  uint64
}

func (t team) Sum() uint64 {
	return t.ID
}
func (b beacon) Sum() uint64 {
	return b.ID
}
func (t *team) Hash(h *hasher) uint64 {
	if t.hash == 0 {
		h.Hash(t.ID)
		h.Hash(t.Name)
		h.Hash(t.Logo)
		h.Hash(t.Color)
		h.Hash(t.Offense)
		h.Hash(t.Minimal)
		t.hash = h.Segment()
	}
	t.total = t.hash
	for i := range t.Hosts {
		t.total += t.Hosts[i].Hash(h)
	}
	for i := range t.Beacons {
		t.total += t.Beacons[i].Hash(h)
	}
	t.total += t.Flags.Hash(h)
	t.total += t.Score.Hash(h)
	t.total += t.Tickets.Hash(h)
	return t.hash
}
func (b *beacon) Hash(h *hasher) uint64 {
	if b.hash == 0 {
		h.Hash(b.ID)
		h.Hash(b.Team)
		h.Hash(b.Color)
		b.hash = h.Segment()
	}
	return b.hash
}
func (t team) Compare(p *planner, o team) {
	if o.ID == 0 {
		p.DeltaValue("team-t"+strconv.FormatUint(t.ID, 10), "", "team")
	} else {
		p.Value("team-t"+strconv.FormatUint(t.ID, 10), "", "team")
	}
	p.Prefix(p.prefix + "-team-t" + strconv.FormatUint(t.ID, 10))
	if o.hash == t.hash {
		p.Value("beacon", "", "team-beacon")
		p.Value("beacon-con", "", "team-beacon-container")
		p.Value("logo", "", "team-logo")
		p.Value("name", "", "team-name")
		p.Value("host", "", "team-host")
		p.Value("score", "", "team-score")
		p.Value("name-name", t.Name, "team-name-div")
		p.Property("logo", t.Color, "background-color")
		p.Property("logo", "url('"+t.Logo+"')", "background-image")
		p.Property("", t.Color, "border-color")
		if t.Offense {
			p.Property("", "+offense", "class")
		} else {
			p.Property("", "-offense", "class")
		}
		if t.Minimal {
			p.Property("", "+mini", "class")
		} else {
			p.Property("", "-mini", "class")
		}
		t.Score.Compare(p, o.Score)
		t.Flags.Compare(p, o.Flags)
		t.Tickets.Compare(p, o.Tickets)
		if o.total == t.total {
			for i := range t.Hosts {
				t.Hosts[i].Compare(p, o.Hosts[i])
			}
			for i := range t.Beacons {
				t.Beacons[i].Compare(p, o.Beacons[i])
			}
		}
	} else {
		p.DeltaValue("beacon", "", "team-beacon")
		p.DeltaValue("beacon-con", "", "team-beacon-container")
		p.DeltaValue("logo", "", "team-logo")
		p.DeltaValue("name", "", "team-name")
		p.DeltaValue("host", "", "team-host")
		p.DeltaValue("score", "", "team-score")
		p.DeltaValue("name-name", t.Name, "team-name-div")
		p.DeltaProperty("logo", t.Color, "background-color")
		p.DeltaProperty("logo", "url('"+t.Logo+"')", "background-image")
		p.DeltaProperty("", t.Color, "border-color")
		if t.Offense {
			p.DeltaProperty("", "+offense", "class")
		} else {
			p.DeltaProperty("", "-offense", "class")
		}
		if t.Minimal {
			p.DeltaProperty("", "+mini", "class")
		} else {
			p.DeltaProperty("", "-mini", "class")
		}
	}
	if o.ID == 0 || o.total != t.total {
		y, u := make(compare), make(compare)
		t.Score.Compare(p, o.Score)
		t.Flags.Compare(p, o.Flags)
		t.Tickets.Compare(p, o.Tickets)
		for i := range o.Hosts {
			y.One(o.Hosts[i])
		}
		for i := range o.Beacons {
			u.One(o.Beacons[i])
		}
		for i := range t.Hosts {
			y.Two(t.Hosts[i])
		}
		for i := range t.Beacons {
			u.Two(t.Beacons[i])
		}
		for k, v := range y {
			switch {
			case !v.Second():
				p.Remove("host-h" + strconv.FormatUint(k, 10))
			case !v.First():
				v.B.(host).Compare(p, emptyHost)
			default:
				v.B.(host).Compare(p, v.A.(host))
			}
		}
		for k, v := range u {
			switch {
			case !v.Second():
				p.Remove("beacon-con-b" + strconv.FormatUint(k, 10))
			case !v.First():
				v.B.(beacon).Compare(p, emptyBeacon)
			default:
				v.B.(beacon).Compare(p, v.A.(beacon))
			}
		}
	}
	p.rollbackPrefix()
}
func (b beacon) Compare(p *planner, o beacon) {
	if o.ID == 0 {
		p.DeltaValue("beacon-con-b"+strconv.FormatUint(b.ID, 10), "", "beacon")
	} else {
		p.Value("beacon-con-b"+strconv.FormatUint(b.ID, 10), "", "beacon")
	}
	p.Prefix(p.prefix + "-beacon-con-b" + strconv.FormatUint(b.ID, 10))
	if o.hash == b.hash {
		p.Property("", b.Team, "tid")
		p.Property("", b.Color, "background")
		p.rollbackPrefix()
		return
	}
	p.DeltaProperty("", b.Team, "tid")
	p.DeltaProperty("", b.Color, "background")
	p.rollbackPrefix()
}
