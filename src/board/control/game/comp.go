package game

import (
	"fmt"
	"strconv"
)

// Update is a message struct that will be sent to clients to understand which
// objects need to be created or updated when the board information changes.
type Update struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Class  string `json:"class,omitempty"`
	Value  string `json:"value,omitempty"`
	Remove bool   `json:"remove"`
}
type planner struct {
	Delta  []*Update
	Create []*Update

	prefix  string
	lprefix []string
}
type comparable struct {
	c1, c2 interface{}
}

func (p *planner) rollbackPrefix() {
	p.prefix, p.lprefix = p.lprefix[len(p.lprefix)-1], p.lprefix[:len(p.lprefix)-1]
}
func printStr(v interface{}) string {
	var s string
	switch v.(type) {
	case string:
		s = v.(string)
	case int:
		s = strconv.Itoa(v.(int))
	case uint:
		s = strconv.Itoa(int(v.(uint)))
	case int8:
		s = strconv.Itoa(int(v.(int8)))
	case uint8:
		s = strconv.Itoa(int(v.(uint8)))
	case int16:
		s = strconv.Itoa(int(v.(int16)))
	case uint16:
		s = strconv.Itoa(int(v.(uint16)))
	case int32:
		s = strconv.Itoa(int(v.(int32)))
	case uint32:
		s = strconv.Itoa(int(v.(uint32)))
	case int64:
		s = strconv.Itoa(int(v.(int64)))
	case uint64:
		s = strconv.Itoa(int(v.(uint64)))
	case float32, float64:
		s = fmt.Sprintf("%.02f", v)
	default:
		s = fmt.Sprintf("%s", v)
	}
	return s
}
func (p *planner) setPrefix(s string) {
	p.lprefix = append(p.lprefix, p.prefix)
	p.prefix = s
}
func (p *planner) setRemove(id interface{}) {
	i := &Update{
		ID:     printStr(id),
		Remove: true,
	}
	if len(i.ID) == 0 {
		i.ID = p.prefix
	} else if len(p.prefix) > 0 {
		i.ID = fmt.Sprintf("%s-%s", p.prefix, i.ID)
	}
	p.Delta = append(p.Delta, i)
}
func (p *planner) setValue(id, v interface{}, c string) {
	i := &Update{
		ID:    printStr(id),
		Value: printStr(v),
		Class: c,
	}
	if len(i.ID) == 0 {
		i.ID = p.prefix
	} else if len(p.prefix) > 0 {
		i.ID = fmt.Sprintf("%s-%s", p.prefix, i.ID)
	}
	p.Create = append(p.Create, i)
}
func (p *planner) setProperty(id, v interface{}, s string) {
	i := &Update{
		ID:    printStr(id),
		Name:  s,
		Value: printStr(v),
	}
	if len(i.ID) == 0 {
		i.ID = p.prefix
	} else if len(p.prefix) > 0 {
		i.ID = fmt.Sprintf("%s-%s", p.prefix, i.ID)
	}
	p.Create = append(p.Create, i)
}
func (p *planner) setDeltaValue(id, v interface{}, c string) {
	i := &Update{
		ID:    printStr(id),
		Value: printStr(v),
		Class: c,
	}
	if len(i.ID) == 0 {
		i.ID = p.prefix
	} else if len(p.prefix) > 0 {
		i.ID = fmt.Sprintf("%s-%s", p.prefix, i.ID)
	}
	p.Delta = append(p.Delta, i)
	p.Create = append(p.Create, i)
}
func (p *planner) setDeltaProperty(id, v interface{}, s string) {
	i := &Update{
		ID:    printStr(id),
		Name:  s,
		Value: printStr(v),
	}
	if len(i.ID) == 0 {
		i.ID = p.prefix
	} else if len(p.prefix) > 0 {
		i.ID = fmt.Sprintf("%s-%s", p.prefix, i.ID)
	}
	p.Delta = append(p.Delta, i)
	p.Create = append(p.Create, i)
}
