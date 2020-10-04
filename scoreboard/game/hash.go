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
	"errors"
	"fmt"
	"hash"
	"math"
	"sync"

	"blainsmith.com/go/seahash"
)

var bufs = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 8)
		return &b
	},
}

type hasher struct {
	h, s hash.Hash64
}

func (h *hasher) Reset() {
	if h.h != nil {
		h.h.Reset()
	}
	if h.s != nil {
		h.s.Reset()
	}
}
func (h hasher) Sum64() uint64 {
	if h.h == nil {
		return 0
	}
	return h.h.Sum64()
}
func (h *hasher) Segment() uint64 {
	if h.s == nil {
		return 0
	}
	v := h.s.Sum64()
	h.s.Reset()
	return v
}
func (h *hasher) Hash(v interface{}) error {
	b := *bufs.Get().(*[]byte)
	_ = b[7]
	switch i := v.(type) {
	case bool:
		if i {
			b[0] = 1
		} else {
			b[0] = 0
		}
		h.Write(b[:1])
	case []byte:
		h.Write(i)
	case string:
		h.Write([]byte(i))
	case float32:
		n := math.Float32bits(i)
		b[0], b[1] = byte(n>>24), byte(n>>16)
		b[2], b[3] = byte(n>>8), byte(n)
		h.Write(b[:4])
	case float64:
		n := math.Float64bits(i)
		b[0], b[1] = byte(n>>56), byte(n>>48)
		b[2], b[3] = byte(n>>40), byte(n>>32)
		b[4], b[5] = byte(n>>24), byte(n>>16)
		b[6], b[7] = byte(n>>8), byte(n)
		h.Write(b)
	case int8:
		b[0] = uint8(i)
		h.Write(b[:1])
	case uint8:
		b[0] = i
		h.Write(b[:1])
	case int16:
		b[0], b[1] = byte(i>>8), byte(i)
		h.Write(b[:2])
	case uint16:
		b[0], b[1] = byte(i>>8), byte(i)
		h.Write(b[:2])
	case int32:
		b[0], b[1] = byte(i>>24), byte(i>>16)
		b[2], b[3] = byte(i>>8), byte(i)
		h.Write(b[:4])
	case uint32:
		b[0], b[1] = byte(i>>24), byte(i>>16)
		b[2], b[3] = byte(i>>8), byte(i)
		h.Write(b[:4])
	case int64:
		b[0], b[1] = byte(i>>56), byte(i>>48)
		b[2], b[3] = byte(i>>40), byte(i>>32)
		b[4], b[5] = byte(i>>24), byte(i>>16)
		b[6], b[7] = byte(i>>8), byte(i)
		h.Write(b)
	case uint64:
		b[0], b[1] = byte(i>>56), byte(i>>48)
		b[2], b[3] = byte(i>>40), byte(i>>32)
		b[4], b[5] = byte(i>>24), byte(i>>16)
		b[6], b[7] = byte(i>>8), byte(i)
		h.Write(b)
	case int:
		b[0], b[1] = byte(i>>56), byte(i>>48)
		b[2], b[3] = byte(i>>40), byte(i>>32)
		b[4], b[5] = byte(i>>24), byte(i>>16)
		b[6], b[7] = byte(i>>8), byte(i)
		h.Write(b)
	case uint:
		b[0], b[1] = byte(i>>56), byte(i>>48)
		b[2], b[3] = byte(i>>40), byte(i>>32)
		b[4], b[5] = byte(i>>24), byte(i>>16)
		b[6], b[7] = byte(i>>8), byte(i)
		h.Write(b)
	case fmt.Stringer:
		h.Write([]byte(v.(fmt.Stringer).String()))
	default:
		bufs.Put(&b)
		return errors.New("cannot hash the requested type: " + fmt.Sprintf("%T", v))
	}
	bufs.Put(&b)
	return nil
}
func (h *hasher) Write(b []byte) (int, error) {
	if h.h == nil {
		h.h = seahash.New()
	}
	if h.s == nil {
		h.s = seahash.New()
	}
	h.s.Write(b)
	return h.h.Write(b)
}
