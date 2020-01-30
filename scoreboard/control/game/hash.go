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
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"math"
	"sync"

	"blainsmith.com/go/seahash"
)

var (
	// ErrCannotSum is an error returned by the function 'Add'. This is returned when the passed
	// interface is not a primitive type.
	ErrCannotSum = errors.New("cannot hash the requested type")

	bufs = &sync.Pool{
		New: func() interface{} {
			return make([]byte, 8)
		},
	}
	hashers = &sync.Pool{
		New: func() interface{} {
			return new(Hasher)
		},
	}
)

// Hasher is a struct that represents a segmented
// hashing mechanism in a 32bit hash format.
type Hasher struct {
	h, s hash.Hash64
}

// Reset sets both the Segment and internal hashers to zero.
func (h *Hasher) Reset() {
	if h.h != nil {
		h.h.Reset()
	}
	if h.s != nil {
		h.s.Reset()
	}
}

// Sum64 returns the hash value of the internal hasher.
func (h Hasher) Sum64() uint64 {
	if h.h == nil {
		return 0
	}
	return h.h.Sum64()
}

// Segment returns the hash value of the Segment hasher and resets it for reuse.
func (h *Hasher) Segment() uint64 {
	if h.s == nil {
		return 0
	}
	v := h.s.Sum64()
	h.s.Reset()
	return v
}

// Hash attempts to identify and convert the interface to a hashable type before
// adding using the 'Sum' function. IF the type is not a hashable type, the error 'ErrCannotSum'
// will be returned.
func (h *Hasher) Hash(v interface{}) error {
	b := bufs.Get().([]byte)
	switch v.(type) {
	case bool:
		if v.(bool) {
			b[0] = 1
		} else {
			b[0] = 0
		}
		h.Write(b[:1])
	case []byte:
		h.Write(v.([]byte))
	case string:
		h.Write([]byte(v.(string)))
	case float32:
		f := math.Float32bits(v.(float32))
		binary.BigEndian.PutUint32(b, f)
		h.Write(b[:4])
	case float64:
		f := math.Float64bits(v.(float64))
		binary.BigEndian.PutUint64(b, f)
		h.Write(b)
	case int8:
		b[0] = uint8(v.(int8))
		h.Write(b[:1])
	case uint8:
		b[0] = v.(uint8)
		h.Write(b[:1])
	case int16:
		binary.BigEndian.PutUint16(b, uint16(v.(int16)))
		h.Write(b[:2])
	case uint16:
		binary.BigEndian.PutUint16(b, v.(uint16))
		h.Write(b[:2])
	case int32:
		binary.BigEndian.PutUint32(b, uint32(v.(int32)))
		h.Write(b[:4])
	case uint32:
		binary.BigEndian.PutUint32(b, v.(uint32))
		h.Write(b[:4])
	case int64:
		binary.BigEndian.PutUint64(b, uint64(v.(int64)))
		h.Write(b)
	case uint64:
		binary.BigEndian.PutUint64(b, v.(uint64))
		h.Write(b)
	case int:
		binary.BigEndian.PutUint64(b, uint64(v.(int)))
		h.Write(b)
	case uint:
		binary.BigEndian.PutUint64(b, uint64(v.(uint)))
		h.Write(b)
	case fmt.Stringer:
		h.Write([]byte(v.(fmt.Stringer).String()))
	default:
		bufs.Put(b)
		return fmt.Errorf("type %T: %w", v, ErrCannotSum)
	}
	bufs.Put(b)
	return nil
}

// Write is an alias to Sum and is used to fit the io.Writer interface.
func (h *Hasher) Write(b []byte) (int, error) {
	if h.h == nil {
		h.h = seahash.New()
	}
	if h.s == nil {
		h.s = seahash.New()
	}
	h.s.Write(b)
	return h.h.Write(b)
}
