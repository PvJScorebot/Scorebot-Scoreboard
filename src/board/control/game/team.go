package game

import (
	"fmt"
)

// Temparory URL for Team Images
const gif = "https://media3.giphy.com/media/3ohze4mdptnmdtyXgk/giphy.gif?cid=790b76115d2f3d255151467732f04b78&rid=giphy.gif"

// Team is a struct that repersents a Team object on the
// Scoreboard
type Team struct {
	ID      int64        `json:"id"`
	Name    string       `json:"name"`
	Logo    string       `json:"logo"`
	Color   string       `json:"color"`
	Flags   *ScoreFlag   `json:"flags"`
	Hosts   []*Host      `json:"hosts"`
	Score   *Score       `json:"score"`
	Tickets *ScoreTicket `json:"tickets"`
	Offense bool         `json:"offense"`
	Minimal bool         `json:"minimal"`
	Beacons []*Beacon    `json:"beacons"`

	hash  uint32
	total uint32
}

// Beacon is a struct that repersents a compromise on a host.
type Beacon struct {
	Team  int64  `json:"team"`
	Color string `json:"color"`

	hash uint32
}

func (t *Team) getHash(h *Hasher) uint32 {
	if t.hash == 0 {
		h.Hash(t.ID)
		h.Hash(t.Name)
		h.Hash(t.Logo)
		h.Hash(t.Color)
		h.Hash(t.Offense)
		h.Hash(t.Minimal)
		t.hash = h.Segment()
	}
	/*t.Beacons = append(t.Beacons, []*Beacon{
		&Beacon{Team: 1, Color: "rgb(255, 0, 255)"},
		&Beacon{Team: 2, Color: "rgb(255, 255, 0)"},
		&Beacon{Team: 3, Color: "rgb(255, 0    0)"},
		&Beacon{Team: 4, Color: "rgb(0, 255, 255)"},
		&Beacon{Team: 6, Color: "rgb(0,   0, 255)"},
		&Beacon{Team: 5, Color: "rgb(0,   0, 255)"},
	}...)

	z := make([]*Beacon, 100)
	for i := range z {
		z[i] = &Beacon{
			Team:  int64(i),
			Color: fmt.Sprintf("rgb(%d, %d, %d)", rand.Intn(255), rand.Intn(255), rand.Intn(255)),
		}
	}
	t.Beacons = append(t.Beacons, z...)*/

	t.total = t.hash
	for i := range t.Hosts {
		t.total += t.Hosts[i].getHash(h)
	}
	for i := range t.Beacons {
		t.total += t.Beacons[i].getHash(h)
	}
	t.total += t.Flags.getHash(h)
	t.total += t.Score.getHash(h)
	t.total += t.Tickets.getHash(h)
	return t.hash
}
func (b *Beacon) getHash(h *Hasher) uint32 {
	if b.hash == 0 {
		h.Hash(b.Team)
		h.Hash(b.Color)
		b.hash = h.Segment()
	}
	return b.hash
}
func (t *Team) getDifference(p *planner, old *Team) {
	if old == nil {
		p.setDeltaValue(fmt.Sprintf("team-t%d", t.ID), "", "team")
	} else {
		p.setValue(fmt.Sprintf("team-t%d", t.ID), "", "team")
	}
	p.setPrefix(fmt.Sprintf("%s-team-t%d", p.prefix, t.ID))
	if old != nil && old.hash == t.hash {
		p.setValue("beacon", "", "team-beacon")
		p.setValue("beacon-con", "", "team-beacon-container")
		p.setValue("logo", "", "team-logo")
		p.setValue("name", "", "team-name")
		p.setValue("host", "", "team-host")
		p.setValue("score", "", "team-score")
		p.setValue("name", t.Name, "team-name")
		p.setProperty("logo", t.Color, "background-color")
		p.setProperty("logo", fmt.Sprintf("url('%s')", gif), "background-image")
		p.setProperty("", t.Color, "border-color")
		if t.Offense {
			p.setProperty("", "+offense", "class")
		} else {
			p.setProperty("", "-offense", "class")
		}
		if t.Minimal {
			p.setProperty("", "+mini", "class")
		} else {
			p.setProperty("", "-mini", "class")
		}
		t.Score.getDifference(p, old.Score)
		t.Flags.getDifference(p, old.Flags)
		t.Tickets.getDifference(p, old.Tickets)
		if old.total == t.total {
			for i := range t.Hosts {
				t.Hosts[i].getDifference(p, old.Hosts[i])
			}
			for i := range t.Beacons {
				t.Beacons[i].getDifference(p, old.Beacons[i])
			}
		}
	} else {
		p.setDeltaValue("beacon", "", "team-beacon")
		p.setDeltaValue("beacon-con", "", "team-beacon-container")
		p.setDeltaValue("logo", "", "team-logo")
		p.setDeltaValue("name", "", "team-name")
		p.setDeltaValue("host", "", "team-host")
		p.setDeltaValue("score", "", "team-score")
		p.setDeltaValue("name", t.Name, "team-name")
		p.setDeltaProperty("logo", t.Color, "background-color")
		p.setDeltaProperty("logo", fmt.Sprintf("url('%s')", gif), "background-image")
		p.setDeltaProperty("", t.Color, "border-color")
		if t.Offense {
			p.setDeltaProperty("", "+offense", "class")
		} else {
			p.setDeltaProperty("", "-offense", "class")
		}
		if t.Minimal {
			p.setDeltaProperty("", "+mini", "class")
		} else {
			p.setDeltaProperty("", "-mini", "class")
		}
	}
	if old == nil || old.total != t.total {
		c := make(map[int64]*comparable)
		x := make(map[int64]*comparable)
		if old != nil {
			t.Score.getDifference(p, old.Score)
			t.Flags.getDifference(p, old.Flags)
			t.Tickets.getDifference(p, old.Tickets)
			for i := range old.Hosts {
				c[old.Hosts[i].ID] = &comparable{c1: old.Hosts[i]}
			}
			for i := range old.Beacons {
				x[old.Beacons[i].Team] = &comparable{c1: old.Beacons[i]}
			}
		} else {
			t.Score.getDifference(p, nil)
			t.Flags.getDifference(p, nil)
			t.Tickets.getDifference(p, nil)
		}
		for i := range t.Hosts {
			v, ok := c[t.Hosts[i].ID]
			if !ok {
				v = &comparable{}
				c[t.Hosts[i].ID] = v
			}
			v.c2 = t.Hosts[i]
		}
		for i := range t.Beacons {
			v, ok := x[t.Beacons[i].Team]
			if !ok {
				v = &comparable{}
				x[t.Beacons[i].Team] = v
			}
			v.c2 = t.Beacons[i]
		}
		for k, v := range c {
			if v.c2 == nil {
				p.setRemove(fmt.Sprintf("h%d", k))
			} else if v.c1 == nil {
				v.c2.(*Host).getDifference(p, nil)
			} else {
				v.c2.(*Host).getDifference(p, v.c1.(*Host))
			}
		}
		for k, v := range x {
			if v.c2 == nil {
				p.setRemove(fmt.Sprintf("beacon-con-b%d", k))
			} else if v.c1 == nil {
				v.c2.(*Beacon).getDifference(p, nil)
			} else {
				v.c2.(*Beacon).getDifference(p, v.c1.(*Beacon))
			}
		}
	}
	p.rollbackPrefix()
}
func (b *Beacon) getDifference(p *planner, old *Beacon) {
	if old == nil {
		p.setDeltaValue(fmt.Sprintf("beacon-con-b%d", b.Team), "", "beacon")
	} else {
		p.setValue(fmt.Sprintf("beacon-con-b%d", b.Team), "", "beacon")
	}
	p.setPrefix(fmt.Sprintf("%s-beacon-con-b%d", p.prefix, b.Team))
	if old != nil && old.hash == b.hash {
		p.setProperty("", b.Team, "iid")
		p.setProperty("", b.Color, "background")
	} else {
		p.setDeltaProperty("", b.Team, "iid")
		p.setDeltaProperty("", b.Color, "background")
	}
	p.rollbackPrefix()
}
