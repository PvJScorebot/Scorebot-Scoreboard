// Copyright(C) 2020 - 2023 iDigitalFlame
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
	"sync"
	"unsafe"
)

const (
	fnvPrime = 1099511628211
	fnvStart = 14695981039346656037
)

var bufs = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 8)
		return &b
	},
}

type hasher struct {
	h, s uint64
}
type stringer interface {
	String() string
}

func (h *hasher) Reset() {
	h.h, h.s = fnvStart, fnvStart
}
func (h hasher) Sum64() uint64 {
	return h.h
}
func (h *hasher) Write(b []byte) {
	if h.h == 0 {
		h.h = fnvStart
	}
	if h.s == 0 {
		h.s = fnvStart
	}
	h.s = updateFnv(h.s, b)
	h.h = updateFnv(h.h, b)
}
func (h *hasher) Segment() uint64 {
	v := h.s
	h.s = fnvStart
	return v
}
func updateFnv(h uint64, b []byte) uint64 {
	for i := range b {
		h *= fnvPrime
		h ^= uint64(b[i])
	}
	return h
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
		n := *(*uint32)(unsafe.Pointer(&i))
		b[0], b[1] = byte(n>>24), byte(n>>16)
		b[2], b[3] = byte(n>>8), byte(n)
		h.Write(b[:4])
	case float64:
		n := *(*uint64)(unsafe.Pointer(&i))
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
	case stringer:
		h.Write([]byte(v.(stringer).String()))
	default:
		bufs.Put(&b)
		return errors.New("cannot hash the requested type")
	}
	bufs.Put(&b)
	return nil
}
