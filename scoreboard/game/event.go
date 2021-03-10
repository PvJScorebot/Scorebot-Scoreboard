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

import "strconv"

var (
	emptyTweet  tweet
	emptyEvents events
)

type event struct {
	Data map[string]string `json:"data"`
	ID   uint64            `json:"id"`
	Type uint8             `json:"type"`
}
type events struct {
	Window  event
	Current []event

	hash uint64
}

func (e event) Sum() uint64 {
	return e.ID
}
func (e *events) Hash(h *hasher) uint64 {
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
func compareTweet(p *planner, n, o tweet) {
	if o.ID == 0 {
		p.DeltaValue("tweet-t"+strconv.FormatUint(n.ID, 64), "", "tweet")
	} else {
		p.Value("tweet-t"+strconv.FormatUint(n.ID, 64), "", "tweet")
	}
	p.Prefix(p.prefix + "-tweet-t" + strconv.FormatUint(n.ID, 64))
	if o.ID > 0 {
		p.Value("pic", "", "tweet-pic")
		p.Property("pic-img", "url('"+n.UserPhoto+"')", "background-image")
		p.Value("user", n.User, "tweet-user")
		p.Value("user-name", n.UserName, "tweet-username")
		p.Value("user-content", n.Text, "tweet-content")
		p.Value("image", "", "tweet-media")
		for x := range n.Images {
			p.Value("image-"+strconv.Itoa(x), "", "tweet-image")
			p.Property("image-"+strconv.Itoa(x), "url('"+n.Images[x]+"')", "background-image")
		}
		p.rollbackPrefix()
		return
	}
	p.DeltaValue("pic", "", "tweet-pic")
	p.DeltaProperty("pic-img", "url('"+n.UserPhoto+"')", "background-image")
	p.DeltaValue("user", n.User, "tweet-user")
	p.DeltaValue("user-name", n.UserName, "tweet-username")
	p.DeltaValue("user-content", n.Text, "tweet-content")
	p.DeltaValue("image", "", "tweet-media")
	for x := range n.Images {
		p.DeltaValue("image-"+strconv.Itoa(x), "", "tweet-image")
		p.DeltaProperty("image-"+strconv.Itoa(x), "url('"+n.Images[x]+"')", "background-image")
	}
	p.rollbackPrefix()
}
func (g *game) hashTweets(h *hasher) uint64 {
	if g.tweets == 0 {
		for i := range g.Tweets {
			h.Hash(g.Tweets[i].ID)
		}
		g.tweets = h.Segment()
	}
	return g.tweets
}
func (g game) compareTweets(p *planner, o *game) {
	if o != nil && o.tweets == g.tweets {
		for i := range g.Tweets {
			compareTweet(p, g.Tweets[i], o.Tweets[i])
		}
		return
	}
	c := make(compare)
	if o != nil {
		for i := range o.Tweets {
			c.One(o.Tweets[i])
		}
	}
	for i := range g.Tweets {
		c.Two(g.Tweets[i])
	}
	for k, v := range c {
		switch {
		case !v.Second():
			p.Remove("tweet-t" + strconv.FormatUint(k, 64))
		case !v.First():
			compareTweet(p, v.B.(tweet), emptyTweet)
		default:
			compareTweet(p, v.B.(tweet), v.A.(tweet))
		}
	}
}
func (e *events) Compare(p *planner, o events) {
	if o.hash == 0 {
		e.Window = o.Window
	}
	if o.hash == e.hash {
		for i := range e.Current {
			p.Event(e.Current[i].ID, e.Current[i].Type, e.Current[i].Data)
		}
		return
	}
	c := make(compare)
	for i := range o.Current {
		c.One(o.Current[i])
	}
	for i := range e.Current {
		c.Two(e.Current[i])
	}
	for k, v := range c {
		if !v.Second() {
			p.RemoveEvent(k, v.A.(event).Type)
			continue
		}
		if v.B.(event).Type > 0 {
			e.setWindowEvent(p, v.B.(event))
		}
		if !v.First() {
			p.DeltaEvent(k, v.B.(event).Type, v.B.(event).Data)
			continue
		}
		p.Event(k, v.B.(event).Type, v.B.(event).Data)
	}
}
func (e *events) setWindowEvent(p *planner, w event) {
	if w.Type <= 0 || e.Window.ID == w.ID {
		return
	}
	if e.Window.ID > 0 {
		p.RemoveEvent(w.ID, w.Type)
	}
	e.Window = w
}
