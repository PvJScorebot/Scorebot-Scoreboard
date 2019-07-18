package game

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Service State constants.
const (
	Red    State = 0x2
	Green  State = 0x0
	Yellow State = 0x1
)

// Service Protocol constants.
const (
	TCP  Protocol = 0x0
	UDP  Protocol = 0x1
	ICMP Protocol = 0x2
)

// State repersents the state of a service using an integer value
type State uint8

// Protocol is an integer value that repersents a Networking protocol
type Protocol uint8

// Host is a struct that repersents all the properties of a Host on the
// Scoreboard.
type Host struct {
	ID       int64      `json:"id"`
	Name     string     `json:"name"`
	Online   bool       `json:"online"`
	Services []*Service `json:"services"`

	hash  uint32
	total uint32
}

// Service is a struct that repersents all the properties of a Service on the
// Scoreboard.
type Service struct {
	ID       int64    `json:"id"`
	Port     uint16   `json:"port"`
	State    State    `json:"status"`
	Bonus    bool     `json:"bool"`
	Protocol Protocol `json:"protocol"`

	hash uint32
}

// String returns the proper name of this Service State.
func (s State) String() string {
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

// String returns the proper name of this Service Protocol.
func (p Protocol) String() string {
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
func (h *Host) getHash(i *Hasher) uint32 {
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
func (s *Service) getHash(h *Hasher) uint32 {
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

// UnmarshalJSON attempts to get the Service State value from a simple JSON string.
func (s *State) UnmarshalJSON(b []byte) error {
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

// UnmarshalJSON attempts to get the Service Protocol value from a simple JSON string.
func (p *Protocol) UnmarshalJSON(b []byte) error {
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
func (h *Host) getDifference(p *planner, old *Host) {
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
				v.c2.(*Service).getDifference(p, nil)
			} else {
				v.c2.(*Service).getDifference(p, v.c1.(*Service))
			}
		}
	}
	p.rollbackPrefix()
}
func (s *Service) getDifference(p *planner, old *Service) {
	if old == nil {
		p.setDeltaValue(fmt.Sprintf("s%d", s.ID), "", "service")
	} else {
		p.setValue(fmt.Sprintf("s%d", s.ID), "", "service")
	}
	p.setPrefix(fmt.Sprintf("%s-s%d", p.prefix, s.ID))
	if old != nil && old.hash == s.hash {
		p.setValue("port", s.Port, "service-port")
		p.setValue("protocol", s.Protocol.String(), "service-protocol")
		p.setProperty("", s.State.String(), "background-color")
		if s.Bonus {
			p.setProperty("", "+bonus", "class")
		} else {
			p.setProperty("", "-bonus", "class")
		}
	} else {
		p.setDeltaValue("port", s.Port, "service-port")
		p.setDeltaValue("protocol", s.Protocol.String(), "service-protocol")
		p.setDeltaProperty("", s.State.String(), "background-color")
		if s.Bonus {
			p.setDeltaProperty("", "+bonus", "class")
		} else {
			p.setDeltaProperty("", "-bonus", "class")
		}
	}
	p.rollbackPrefix()
}
