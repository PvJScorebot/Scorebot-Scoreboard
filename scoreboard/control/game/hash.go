package game

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"sync"
)

var (
	// ErrCannotSum is an error returned by the function 'Add'. This is returned when the passed
	// interface is not a primitive type.
	ErrCannotSum = errors.New("cannot hash sum requested type")

	bufPool = &sync.Pool{
		New: func() interface{} {
			return make([]byte, 8)
		},
	}
)

// Hasher is a struct that represents a segmented
// hashing mechanism in a 32bit hash format.
type Hasher struct {
	h hash.Hash32
	s hash.Hash32
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

// Sum32 returns the hash value of the internal hasher.
func (h *Hasher) Sum32() uint32 {
	if h.h == nil {
		return 0
	}
	return h.h.Sum32()
}

// Segment returns the hash value of the Segment hasher and resets it for reuse.
func (h *Hasher) Segment() uint32 {
	if h.s == nil {
		return 0
	}
	v := h.s.Sum32()
	h.s.Reset()
	return v
}

// Hash attempts to identify and convert the interface to a hashable type before
// adding using the 'Sum' function. IF the type is not a hashable type, the error 'ErrCannotSum'
// will be returned.
func (h *Hasher) Hash(v interface{}) error {
	b := bufPool.Get().([]byte)
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
		bufPool.Put(b)
		return ErrCannotSum
	}
	bufPool.Put(b)
	return nil
}

// Write is an alias to Sum and is used to fit the io.Writer interface.
func (h *Hasher) Write(b []byte) (int, error) {
	if h.h == nil {
		h.h = fnv.New32()
	}
	if h.s == nil {
		h.s = fnv.New32()
	}
	h.s.Write(b)
	return h.h.Write(b)
}
