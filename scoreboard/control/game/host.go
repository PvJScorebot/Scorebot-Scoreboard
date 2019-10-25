package game

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Service State constants.
const (
	Red    state = 0x2
	Green  state = 0x0
	Yellow state = 0x1
)

// Service Protocol constants.
const (
	TCP  protocol = 0x0
	UDP  protocol = 0x1
	ICMP protocol = 0x2
)

type state uint8
type host struct {
	ID       int64      `json:"id"`
	Name     string     `json:"name"`
	Online   bool       `json:"online"`
	Services []*service `json:"services"`

	hash  uint64
	total uint64
}
type protocol uint8
type service struct {
	ID       int64    `json:"id"`
	Port     uint16   `json:"port"`
	State    state    `json:"status"`
	Bonus    bool     `json:"bool"`
	Protocol protocol `json:"protocol"`

	hash uint64
}

func (s state) class() string {
	switch s {
	case Red:
		return "err"
	case Yellow:
		return "warn"
	case Green:
		return "port"
	}
	return "port"
}
func (s state) String() string {
	switch s {
	case Red:
		return "rgb(255, 0, 0)"
	case Yellow:
		return "rgb(173, 164, 21)"
	case Green:
		return "rgb(40, 111, 36)"
	}
	return "rgb(255, 0, 0)"
}
func (p protocol) String() string {
	switch p {
	case TCP:
		return "TCP"
	case UDP:
		return "UDP"
	case ICMP:
		return "ICMP"
	}
	return "Unknown"
}
func (h *host) getHash(i *Hasher) uint64 {
	if h.hash == 0 {
		i.Hash(h.ID)
		i.Hash(h.Name)
		i.Hash(h.Online)
		h.hash = i.Segment()
	}
	h.total = h.hash
	for s := range h.Services {
		h.total += h.Services[s].getHash(i)
	}
	return h.hash
}
func (s *service) getHash(h *Hasher) uint64 {
	if s.hash == 0 {
		h.Hash(s.ID)
		h.Hash(s.Port)
		h.Hash(s.State)
		h.Hash(s.Bonus)
		h.Hash(s.Protocol)
		s.hash = h.Segment()
	}
	return s.hash
}
func (s *state) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch strings.ToLower(v) {
	case "red", "r", "fail":
		*s = Red
	case "yellow", "y", "issue":
		*s = Yellow
	case "green", "g", "good", "ok":
		*s = Green
	default:
		*s = Red
	}
	return nil
}
func (p *protocol) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch strings.ToLower(v) {
	case "tcp", "t":
		*p = TCP
	case "udp", "u":
		*p = UDP
	case "icmp", "i", "p", "ping":
		*p = ICMP
	default:
		*p = TCP
	}
	return nil
}
func (h *host) getDifference(p *planner, old *host) {
	if old == nil {
		p.setDeltaValue(fmt.Sprintf("host-h%d", h.ID), "", "host")
	} else {
		p.setValue(fmt.Sprintf("host-h%d", h.ID), "", "host")
	}
	p.setPrefix(fmt.Sprintf("%s-host-h%d", p.prefix, h.ID))
	if old != nil && old.hash == h.hash {
		p.setValue("name", h.Name, "host-name")
		if h.Online {
			p.setProperty("", "-offline", "class")
		} else {
			p.setProperty("", "+offline", "class")
		}
		if old.total == h.total {
			for i := range h.Services {
				h.Services[i].getDifference(p, old.Services[i])
			}
		}
	} else {
		p.setDeltaValue("name", h.Name, "host-name")
		if h.Online {
			p.setDeltaProperty("", "-offline", "class")
		} else {
			p.setDeltaProperty("", "+offline", "class")
		}
	}
	if old == nil || old.total != h.total {
		c := make(map[int64]*comparable)
		if old != nil {
			for i := range old.Services {
				c[old.Services[i].ID] = &comparable{c1: old.Services[i]}
			}
		}
		for i := range h.Services {
			v, ok := c[h.Services[i].ID]
			if !ok {
				v = &comparable{}
				c[h.Services[i].ID] = v
			}
			v.c2 = h.Services[i]
		}
		for k, v := range c {
			if v.c2 == nil {
				p.setRemove(fmt.Sprintf("s%d", k))
			} else if v.c1 == nil {
				v.c2.(*service).getDifference(p, nil)
			} else {
				v.c2.(*service).getDifference(p, v.c1.(*service))
			}
		}
	}
	p.rollbackPrefix()
}
func (s *service) getDifference(p *planner, old *service) {
	if old == nil {
		p.setDeltaValue(fmt.Sprintf("s%d", s.ID), "", "service")
	} else {
		p.setValue(fmt.Sprintf("s%d", s.ID), "", "service")
	}
	p.setPrefix(fmt.Sprintf("%s-s%d", p.prefix, s.ID))
	if old != nil && old.hash == s.hash {
		p.setValue("port", s.Port, s.State.class())
		p.setValue("protocol", s.Protocol.String(), "service-protocol")
		if s.Bonus {
			p.setProperty("", "+bonus", "class")
		} else {
			p.setProperty("", "-bonus", "class")
		}
		p.setProperty("", s.State.String(), "background-color")
	} else {
		p.setDeltaValue("port", s.Port, s.State.class())
		p.setDeltaValue("protocol", s.Protocol.String(), "service-protocol")
		if s.Bonus {
			p.setDeltaProperty("", "+bonus", "class")
		} else {
			p.setDeltaProperty("", "-bonus", "class")
		}
		p.setDeltaProperty("", s.State.String(), "background-color")
	}
	p.rollbackPrefix()
}
