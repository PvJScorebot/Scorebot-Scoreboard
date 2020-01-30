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

type score struct {
	Total  int64 `json:"total"`
	Health int64 `json:"health"`

	hash uint64
}
type scoreFlag struct {
	Open     uint32 `json:"open"`
	Lost     uint32 `json:"lost"`
	Captured uint32 `json:"captured"`

	hash uint64
}
type scoreTicket struct {
	Open   uint32 `json:"open"`
	Closed uint32 `json:"closed"`

	hash uint64
}

func (s *score) getHash(h *Hasher) uint64 {
	if s.hash == 0 {
		h.Hash(s.Total)
		h.Hash(s.Health)
		s.hash = h.Segment()
	}
	return s.hash
}
func (s *scoreFlag) getHash(h *Hasher) uint64 {
	if s.hash == 0 {
		h.Hash(s.Open)
		h.Hash(s.Lost)
		h.Hash(s.Captured)
		s.hash = h.Segment()
	}
	return s.hash
}
func (s *scoreTicket) getHash(h *Hasher) uint64 {
	if s.hash == 0 {
		h.Hash(s.Open)
		h.Hash(s.Closed)
		s.hash = h.Segment()
	}
	return s.hash
}
func (s score) getDifference(p *planner, old *score) {
	if old != nil && old.hash == s.hash {
		p.setValue("name-total", s.Total, "score-total score")
		p.setValue("score-health", s.Health, "score-health score")
	} else {
		p.setDeltaValue("name-total", s.Total, "score-total score")
		p.setDeltaValue("score-health", s.Health, "score-health score")
	}
}
func (s scoreFlag) getDifference(p *planner, old *scoreFlag) {
	if old != nil && old.hash == s.hash {
		p.setValue("score-fopen", s.Open, "score-flag-open score score-flag")
		p.setValue("score-flost", s.Lost, "score-flag-lost score score-flag")
		p.setValue("score-fcaptured", s.Captured, "score-flag-captured score score-flag")
	} else {
		p.setDeltaValue("score-fpen", s.Open, "score-flag-open score score-flag")
		p.setDeltaValue("score-flost", s.Lost, "score-flag-lost score score-flag")
		p.setDeltaValue("score-fcaptured", s.Captured, "score-flag-captured score score-flag")
	}
}
func (s scoreTicket) getDifference(p *planner, old *scoreTicket) {
	if old != nil && old.hash == s.hash {
		p.setValue("score-topen", s.Open, "score-ticket-open score score-ticket")
		p.setValue("score-tclosed", s.Closed, "score-ticket-closed score score-ticket")
	} else {
		p.setDeltaValue("score-topen", s.Open, "score-ticket-open score score-ticket")
		p.setDeltaValue("score-tclosed", s.Closed, "score-ticket-closed score score-ticket")
	}
}
