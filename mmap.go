package geo

import (
	"errors"
	"io"
	"sort"

	"github.com/tidwall/mmap"
)

var ErrNotFound = errors.New("not found")

type Decoder[T Float] interface {
	Decode([]byte) error
	Size() int // size of struct
	Point() Point[T]
	Less(Point[T]) bool
	JSON(w io.Writer) error
}

type MFile[T Float] struct {
	B []byte
}

type Iter[T Float] struct {
	m *MFile[T]
	d Decoder[T]
}

func (m *MFile[T]) Close() error {
	return mmap.Close(m.B)
}

func (m *Iter[T]) Len() int {
	return len(m.m.B) / m.d.Size()
}

func (m *Iter[T]) IndexPoint(i int) Point[T] {
	off := m.d.Size() * i
	end := off + m.d.Size()
	if err := m.d.Decode(m.m.B[off:end]); err != nil {
		panic(err)
	}
	return m.d.Point()
}

func (m *Iter[T]) Load(i int) {
	off := m.d.Size() * i
	end := off + m.d.Size()
	if err := m.d.Decode(m.m.B[off:end]); err != nil {
		panic(err)
	}
}

func (m *Iter[T]) Less(pt Point[T]) bool {
	return m.d.Less(pt)
}

func (m *Iter[T]) JSON(w io.Writer) {
	m.d.JSON(w)
}

func Mmap[T Float](filename string) (*MFile[T], error) {
	b, err := mmap.Open(filename, false)
	if err != nil {
		return nil, err
	}
	return &MFile[T]{b}, err
}

func (m *MFile[T]) ReadAt(p []byte, i int64) (int, error) {
	if i > int64(len(m.B)) {
		return 0, errors.New("index exceeds file size")
	}
	return copy(p, m.B[i:]), nil
}

func (m *MFile[T]) NewIter(d Decoder[T]) *Iter[T] {
	return &Iter[T]{
		m: m,
		d: d,
	}
}

func (m *Iter[T]) Get(i int) interface{} {
	off := m.d.Size() * i
	end := off + m.d.Size()
	if err := m.d.Decode(m.m.B[off:end]); err != nil {
		panic(err)
	}
	return m.d
}

type Container[T Float] interface {
	ContainsPoint(Point[T]) bool
}

func (m *Iter[T]) Ranger(from, to Point[T], fn func(interface{}), ctr Container[T]) error {
	size := m.Len()
	idx := sort.Search(size, func(i int) bool {
		return from.Less(m.IndexPoint(i))
	})
	if idx == size {
		return ErrNotFound
	}
	for {
		m.Load(idx)
		if !m.Less(to) {
			break
		}
		pt := m.d.Point()
		if between(pt.Lon, from.Lon, to.Lon) {
			if ctr == nil || ctr.ContainsPoint(m.d.Point()) {
				fn(m.d)
			}
		}
		idx++
	}
	return nil
}
