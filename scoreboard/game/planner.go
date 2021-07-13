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
	"fmt"
	"strconv"
)

type delta struct {
	A, B interface{}
}
type update struct {
	Value  interface{}       `json:"value,omitempty"`
	Data   map[string]string `json:"data"`
	ID     string            `json:"id"`
	Name   string            `json:"name,omitempty"`
	Class  string            `json:"class,omitempty"`
	Event  bool              `json:"event"`
	Remove bool              `json:"remove"`
}
type planner struct {
	prefix string
	Delta  []update
	Create []update
	last   []string
}
type comparable interface {
	Sum() uint64
}
type compare map[uint64]delta

func (d delta) First() bool {
	return d.A != nil
}
func (d delta) Second() bool {
	return d.B != nil
}
func (p *planner) Prefix(s string) {
	p.last = append(p.last, p.prefix)
	p.prefix = s
}
func (c compare) One(d comparable) {
	c[d.Sum()] = delta{A: d}
}
func (c compare) Two(d comparable) {
	s := d.Sum()
	v, ok := c[s]
	if !ok {
		c[s] = delta{B: d}
		return
	}
	v.B = d
}
func (p *planner) rollbackPrefix() {
	p.prefix, p.last = p.last[len(p.last)-1], p.last[:len(p.last)-1]
}
func printStr(v interface{}) string {
	var s string
	switch i := v.(type) {
	case string:
		s = i
	case int:
		s = strconv.Itoa(i)
	case uint:
		s = strconv.FormatUint(uint64(i), 10)
	case int8:
		s = strconv.Itoa(int(i))
	case uint8:
		s = strconv.FormatUint(uint64(i), 10)
	case int16:
		s = strconv.Itoa(int(i))
	case uint16:
		s = strconv.FormatUint(uint64(i), 10)
	case int32:
		s = strconv.Itoa(int(i))
	case uint32:
		s = strconv.FormatUint(uint64(i), 10)
	case int64:
		s = strconv.Itoa(int(i))
	case uint64:
		s = strconv.FormatUint(uint64(i), 10)
	case float32:
		s = strconv.FormatFloat(float64(i), 'f', 2, 32)
	case float64:
		s = strconv.FormatFloat(i, 'f', 2, 64)
	default:
		s = fmt.Sprintf("%s", v)
	}
	return s
}
func (p *planner) Remove(i interface{}) {
	u := update{ID: printStr(i), Remove: true}
	if len(u.ID) == 0 {
		u.ID = p.prefix
	} else if len(p.prefix) > 0 {
		u.ID = p.prefix + "-" + u.ID
	}
	p.Delta = append(p.Delta, u)
}
func (p *planner) RemoveEvent(i uint64, t uint8) {
	p.Delta = append(p.Delta, update{
		ID:     strconv.Itoa(int(i)),
		Value:  strconv.Itoa(int(t)),
		Event:  true,
		Remove: true,
	})
}
func (p *planner) Value(i, v interface{}, c string) {
	u := update{ID: printStr(i), Value: printStr(v), Class: c}
	if len(u.ID) == 0 {
		u.ID = p.prefix
	} else if len(p.prefix) > 0 {
		u.ID = p.prefix + "-" + u.ID
	}
	p.Create = append(p.Create, u)
}
func (p *planner) Property(i, v interface{}, s string) {
	u := update{ID: printStr(i), Name: s, Value: printStr(v)}
	if len(u.ID) == 0 {
		u.ID = p.prefix
	} else if len(p.prefix) > 0 {
		u.ID = p.prefix + "-" + u.ID
	}
	p.Create = append(p.Create, u)
}
func (p *planner) DeltaValue(i, v interface{}, c string) {
	u := update{ID: printStr(i), Value: printStr(v), Class: c}
	if len(u.ID) == 0 {
		u.ID = p.prefix
	} else if len(p.prefix) > 0 {
		u.ID = p.prefix + "-" + u.ID
	}
	p.Delta = append(p.Delta, u)
	p.Create = append(p.Create, u)
}
func (p *planner) DeltaProperty(i, v interface{}, s string) {
	u := update{ID: printStr(i), Name: s, Value: printStr(v)}
	if len(u.ID) == 0 {
		u.ID = p.prefix
	} else if len(p.prefix) > 0 {
		u.ID = p.prefix + "-" + u.ID
	}
	p.Delta = append(p.Delta, u)
	p.Create = append(p.Create, u)
}
func (p *planner) Event(i uint64, t uint8, d map[string]string) {
	p.Create = append(p.Create, update{
		ID:    strconv.Itoa(int(i)),
		Data:  d,
		Event: true,
		Value: strconv.Itoa(int(t)),
	})
}
func (p *planner) DeltaEvent(i uint64, t uint8, d map[string]string) {
	u := update{
		ID:    strconv.FormatUint(i, 10),
		Data:  d,
		Event: true,
		Value: strconv.FormatUint(uint64(t), 10),
	}
	p.Delta = append(p.Delta, u)
	p.Create = append(p.Create, u)
}
