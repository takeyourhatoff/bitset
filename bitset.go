// Package bitset provides Bitset, a compact and fast representation for a dense set of positive integer values.
package bitset

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/bits"
)

const maxUint = 1<<bits.UintSize - 1

// Bitset represents a set of positive integers. Memory usage is proportional to the largest integer in the Bitset.
type Bitset struct {
	s []uint
}

// Add adds the integer i to s. Add panics if i is less than zero.
func (s *Bitset) Add(i int) {
	if i < 0 {
		panic("bitset: cannot add non-negative integer to set")
	}
	w, mask := idx(i)
	for j := len(s.s); j <= w; j++ {
		s.s = append(s.s, 0)
	}
	s.s[w] |= mask
}

// Remove removes the integer i from s, or does nothing if i is not already in s.
func (s *Bitset) Remove(i int) {
	if i < 0 {
		// i < 0 cannot bit in set by definition.
		return
	}
	w, mask := idx(i)
	if w < len(s.s) {
		s.s[w] &^= mask
	}
}

// Test returns true if i is in s, false otherwise.
func (s *Bitset) Test(i int) bool {
	w, mask := idx(i)
	if i < 0 || w >= len(s.s) {
		return false
	}
	return s.s[w]&mask != 0
}

// Max returns the value of the maximum integer in s, or -1 if s is empty.
func (s *Bitset) Max() int {
	for n := len(s.s) - 1; n >= 0; n-- {
		if s.s[n] == 0 {
			continue
		}
		return bits.UintSize*(n+1) - bits.LeadingZeros(s.s[n]) - 1
	}
	return -1
}

// And removes integers in s which are not also in ss.
func (s *Bitset) And(ss *Bitset) {
	n := min(len(s.s), len(ss.s))
	for i := 0; i < n; i++ {
		s.s[i] &= ss.s[i]
	}
	s.s = s.s[:n]
}

// AndNot removes integers from s which are also in ss.
func (s *Bitset) AndNot(ss *Bitset) {
	n := min(len(s.s), len(ss.s))
	for i := 0; i < n; i++ {
		s.s[i] &^= ss.s[i]
	}
}

// Or adds integers to s which are in ss.
func (s *Bitset) Or(ss *Bitset) {
	n := min(len(s.s), len(ss.s))
	for i := 0; i < n; i++ {
		s.s[i] |= ss.s[i]
	}
	if len(ss.s) > len(s.s) {
		s.s = append(s.s, ss.s[len(s.s):]...)
	}
}

// XOr adds integers to s which are in ss but not in s, and removes integers in s that are also in ss.
func (s *Bitset) XOr(ss *Bitset) {
	n := min(len(s.s), len(ss.s))
	for i := 0; i < n; i++ {
		s.s[i] ^= ss.s[i]
	}
	if len(ss.s) > len(s.s) {
		s.s = append(s.s, ss.s[len(s.s):]...)
	}
}

// Count returns the number of integers in s.
func (s *Bitset) Count() int {
	var n int
	for i := range s.s {
		n += bits.OnesCount(s.s[i])
	}
	return n
}

// NextAfter returns the smallest integer in s greater than or equal to i or -1 if no such integer exists.
func (s *Bitset) NextAfter(i int) int {
	if i < 0 {
		// There can be no integers in s less than 0 by definition
		i = 0
	}
	mask := uint(maxUint) << (uint(i) % bits.UintSize)
	for j := i / bits.UintSize; j < len(s.s); j++ {
		word := s.s[j] & mask
		mask = maxUint
		if word != 0 {
			return j*bits.UintSize + bits.TrailingZeros(word)
		}
	}
	return -1
}

// Copy returns a copy of s.
func (s *Bitset) Copy() *Bitset {
	n := len(s.s)
	for n > 0 && s.s[n-1] == 0 {
		n--
	}
	ss := new(Bitset)
	ss.s = make([]uint, n)
	copy(ss.s, s.s)
	return ss
}

// String returns a string representation of s.
func (s *Bitset) String() string {
	var buf bytes.Buffer
	buf.WriteRune('[')
	first := true
	for i := s.NextAfter(0); i >= 0; i = s.NextAfter(i + 1) {
		if !first {
			buf.WriteRune(' ')
		}
		fmt.Fprintf(&buf, "%d", i)
		first = false
	}
	buf.WriteRune(']')
	return buf.String()
}

// Bytes returns the set as a bitarray.
// The most significant bit in each byte represents the smallest-index number.
func (s *Bitset) Bytes() []byte {
	const r = bits.UintSize / 8
	b := make([]byte, len(s.s)*r)
	b0 := b
	for _, v := range s.s {
		v = bits.Reverse(v)
		switch bits.UintSize {
		case 32:
			binary.BigEndian.PutUint32(b, uint32(v))
		case 64:
			binary.BigEndian.PutUint64(b, uint64(v))
		default:
			panic("uint is not 32 or 64 bits long")
		}
		b = b[r:]
	}
	for len(b0) > 0 && b0[len(b0)-1] == 0 {
		b0 = b0[:len(b0)-1]
	}
	return b0
}

// FromBytes sets s to the value of data interpreted as a bitarray in the same format as produced by Bytes..
func (s *Bitset) FromBytes(data []byte) {
	const r = bits.UintSize / 8
	if len(data) == 0 {
		s.s = nil
	}
	for len(data)%r != 0 {
		data = append(data, 0)
	}
	s.s = make([]uint, len(data)/r)
	for i := range s.s {
		switch bits.UintSize {
		case 32:
			s.s[i] = uint(binary.BigEndian.Uint32(data))
		case 64:
			s.s[i] = uint(binary.BigEndian.Uint64(data))
		default:
			panic("uint is not 32 or 64 bits long")
		}
		s.s[i] = bits.Reverse(s.s[i])
		data = data[r:]
	}
}

func idx(i int) (w int, mask uint) {
	w = i / bits.UintSize
	mask = 1 << (uint(i) % bits.UintSize)
	return
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
