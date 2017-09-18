// Package bitset provides Set, a compact and fast representation for a dense set of positive integer values.
package bitset

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/bits"
)

const maxUint = 1<<bits.UintSize - 1

// Set represents a set of positive integers. Memory usage is proportional to the largest integer in the Set.
type Set struct {
	s []uint
}

// Add adds the integer i to s. Add panics if i is less than zero.
func (s *Set) Add(i int) {
	if i < 0 {
		panic("bitset: cannot add non-negative integer to set")
	}
	w, mask := idx(i)
	for j := len(s.s); j <= w; j++ {
		s.s = append(s.s, 0)
	}
	s.s[w] |= mask
}

// AddRange adds integers in the interval [low, hi) to the set. AddRange panics if low is less than zero.
func (s *Set) AddRange(low, hi int) {
	if low < 0 {
		panic("bitset: cannot add non-negative integer to set")
	}
	w0, _ := idx(low)
	w1, _ := idx(hi - 1)
	for j := len(s.s); j <= w1; j++ {
		s.s = append(s.s, 0)
	}
	leftMask := uint(maxUint) << (uint(low) % bits.UintSize)
	rightMask := uint(maxUint) >> (uint(bits.UintSize-hi) % bits.UintSize)
	switch {
	case w1-w0 < 0:
		return
	case w1 == w0:
		s.s[w0] |= leftMask & rightMask
	default:
		s.s[w0] |= leftMask
		for i := w0 + 1; i < w1; i++ {
			s.s[i] = maxUint
		}
		s.s[w1] |= rightMask
	}
}

// Remove removes the integer i from s, or does nothing if i is not already in s.
func (s *Set) Remove(i int) {
	if i < 0 {
		// i < 0 cannot bit in set by definition.
		return
	}
	w, mask := idx(i)
	if w < len(s.s) {
		s.s[w] &^= mask
	}
}

// RemoveRange removes integers in the interval [low, hi) from the set.
func (s *Set) RemoveRange(low, hi int) {
	if low < 0 {
		low = 0
	}
	w0, _ := idx(low)
	if w0 >= len(s.s) {
		return
	}
	w1, _ := idx(hi - 1)
	if w1 >= len(s.s) {
		hi = len(s.s) * bits.UintSize
		w1 = len(s.s) - 1
	}
	leftMask := uint(maxUint) << (uint(low) % bits.UintSize)
	rightMask := uint(maxUint) >> (uint(bits.UintSize-hi) % bits.UintSize)
	switch {
	case w1-w0 < 0:
		return
	case w1 == w0:
		s.s[w0] &^= leftMask & rightMask
	default:
		s.s[w0] &^= leftMask
		for i := w0 + 1; i < w1; i++ {
			s.s[i] = 0
		}
		s.s[w1] &^= rightMask
	}
}

// Test returns true if i is in s, false otherwise.
func (s *Set) Test(i int) bool {
	w, mask := idx(i)
	if i < 0 || w >= len(s.s) {
		return false
	}
	return s.s[w]&mask != 0
}

// Max returns the value of the maximum integer in s, or -1 if s is empty.
func (s *Set) Max() int {
	for n := len(s.s) - 1; n >= 0; n-- {
		if s.s[n] == 0 {
			continue
		}
		return bits.UintSize*(n+1) - bits.LeadingZeros(s.s[n]) - 1
	}
	return -1
}

// Intersect removes integers in s which are not also in ss.
func (s *Set) Intersect(ss *Set) {
	n := min(len(s.s), len(ss.s))
	for i := 0; i < n; i++ {
		s.s[i] &= ss.s[i]
	}
	s.s = s.s[:n]
}

// Subtract removes integers from s which are also in ss.
func (s *Set) Subtract(ss *Set) {
	n := min(len(s.s), len(ss.s))
	for i := 0; i < n; i++ {
		s.s[i] &^= ss.s[i]
	}
}

// Union adds integers to s which are in ss.
func (s *Set) Union(ss *Set) {
	n := min(len(s.s), len(ss.s))
	for i := 0; i < n; i++ {
		s.s[i] |= ss.s[i]
	}
	if len(ss.s) > len(s.s) {
		s.s = append(s.s, ss.s[len(s.s):]...)
	}
}

// SymmetricDifference adds integers to s which are in ss but not in s, and removes integers in s that are also in ss.
func (s *Set) SymmetricDifference(ss *Set) {
	n := min(len(s.s), len(ss.s))
	for i := 0; i < n; i++ {
		s.s[i] ^= ss.s[i]
	}
	if len(ss.s) > len(s.s) {
		s.s = append(s.s, ss.s[len(s.s):]...)
	}
}

// Cardinality returns the number of integers in s.
func (s *Set) Cardinality() int {
	var n int
	for i := range s.s {
		n += bits.OnesCount(s.s[i])
	}
	return n
}

func (s *Set) Equal(ss *Set) bool {
	s0, s1 := s.s, ss.s
	for len(s0) > 0 && s0[len(s0)-1] == 0 {
		s0 = s0[:len(s0)-1]
	}
	for len(s1) > 0 && s1[len(s1)-1] == 0 {
		s1 = s1[:len(s1)-1]
	}
	if len(s0) != len(s1) {
		return false
	}
	for i := range s0 {
		if s0[i] != s1[i] {
			return false
		}
	}
	return true
}

// NextAfter returns the smallest integer in s greater than or equal to i or -1 if no such integer exists.
func (s *Set) NextAfter(i int) int {
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
func (s *Set) Copy() *Set {
	n := len(s.s)
	for n > 0 && s.s[n-1] == 0 {
		n--
	}
	ss := new(Set)
	ss.s = make([]uint, n)
	copy(ss.s, s.s)
	return ss
}

// String returns a string representation of s.
func (s *Set) String() string {
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
func (s *Set) Bytes() []byte {
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
func (s *Set) FromBytes(data []byte) {
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
