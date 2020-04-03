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

func (s *score) Hash(h *hasher) uint64 {
	if s.hash == 0 {
		h.Hash(s.Total)
		h.Hash(s.Health)
		s.hash = h.Segment()
	}
	return s.hash
}
func (s *scoreFlag) Hash(h *hasher) uint64 {
	if s.hash == 0 {
		h.Hash(s.Open)
		h.Hash(s.Lost)
		h.Hash(s.Captured)
		s.hash = h.Segment()
	}
	return s.hash
}
func (s score) Compare(p *planner, o score) {
	if o.hash == s.hash {
		p.Value("name-total", s.Total, "score-total score")
		p.Value("score-health", s.Health, "score-health score")
		return
	}
	p.DeltaValue("name-total", s.Total, "score-total score")
	p.DeltaValue("score-health", s.Health, "score-health score")
}
func (s *scoreTicket) Hash(h *hasher) uint64 {
	if s.hash == 0 {
		h.Hash(s.Open)
		h.Hash(s.Closed)
		s.hash = h.Segment()
	}
	return s.hash
}
func (s scoreFlag) Compare(p *planner, o scoreFlag) {
	if o.hash == s.hash {
		p.Value("score-fopen", s.Open, "score-flag-open score score-flag")
		p.Value("score-flost", s.Lost, "score-flag-lost score score-flag")
		p.Value("score-fcaptured", s.Captured, "score-flag-captured score score-flag")
		return
	}
	p.DeltaValue("score-fpen", s.Open, "score-flag-open score score-flag")
	p.DeltaValue("score-flost", s.Lost, "score-flag-lost score score-flag")
	p.DeltaValue("score-fcaptured", s.Captured, "score-flag-captured score score-flag")
}
func (s scoreTicket) Compare(p *planner, o scoreTicket) {
	if o.hash == s.hash {
		p.Value("score-topen", s.Open, "score-ticket-open score score-ticket")
		p.Value("score-tclosed", s.Closed, "score-ticket-closed score score-ticket")
		return
	}
	p.DeltaValue("score-topen", s.Open, "score-ticket-open score score-ticket")
	p.DeltaValue("score-tclosed", s.Closed, "score-ticket-closed score score-ticket")
}
