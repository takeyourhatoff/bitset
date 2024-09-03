package bitset

import (
	"bytes"
	"fmt"
	"iter"
	"math/bits"
	"math/rand"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"testing"
	"testing/quick"
)

// From can be used to iterate over the elements of the set.
func ExampleSet_From() {
	s := new(Set)
	s.Add(2)
	s.Add(42)
	s.Add(13)
	for i := range s.From(0) {
		fmt.Println(i)
	}
	// Output:
	// 2
	// 13
	// 42
}

func ExampleSet_String() {
	s := new(Set)
	s.Add(2)
	s.Add(42)
	s.Add(13)
	fmt.Println(s)
	// Output: [2 13 42]
}

func ExampleSet_Bytes() {
	s := new(Set)
	s.Add(0)
	s.Add(3)
	s.Add(8)
	s.Add(10)
	b := s.Bytes()
	fmt.Printf("%b %b", b[0], b[1])
	// Output: 10010000 10100000
}

func TestAdd_Panic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("b.Add(-1) did not panic")
		} else if err, ok := r.(runtime.Error); ok {
			t.Error(err)
		}
	}()
	new(Set).Add(-1)
}

func TestAddRange_Panic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("b.Add(-1) did not panic")
		} else if err, ok := r.(runtime.Error); ok {
			t.Error(err)
		}
	}()
	new(Set).AddRange(-1, 0)
}

func TestAddAndTest(t *testing.T) {
	f := func(l ascendingInts) bool {
		b := new(Set)
		for _, i := range l {
			b.Add(i)
		}
		min := -10
		max := 10
		if len(l) > 0 {
			max += l[len(l)-1]
		}
		for i := min; i < max; i++ {
			if v := b.Test(i); v != in(i, l) {
				t.Logf("b.Test(%d) = %v, expected %v", i, v, in(i, l))
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestRemove(t *testing.T) {
	f := func(l0, l1 ascendingInts) bool {
		b := new(Set)
		for _, i := range l0 {
			b.Add(i)
		}
		// set l1 to be a subset of l0
		l1 = bitwiseF(func(p, q bool) bool { return p && q }, l0, l1)
		// remove that subset
		for _, i := range l1 {
			b.Remove(i)
		}
		// set l0 to be l0 - l1
		l0 = bitwiseF(func(p, q bool) bool { return p && !q }, l0, l1)
		min := -10
		max := 10
		if len(l0) > 0 {
			max += l0[len(l0)-1]
		}
		for i := min; i < max; i++ {
			if v := b.Test(i); v != in(i, l0) {
				t.Logf("b.Test(%d) = %v, expected %v", i, v, in(i, l0))
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestCopy(t *testing.T) {
	f0 := func(l ascendingInts) string {
		b := new(Set)
		for _, i := range l {
			b.Add(i)
		}
		// Remove the last half to test Copy's trailing zero logic
		lr := l[len(l)/2:]
		for _, i := range lr {
			b.Remove(i)
		}
		return b.String()
	}
	f1 := func(l ascendingInts) string {
		b := new(Set)
		for _, i := range l {
			b.Add(i)
		}
		lr := l[len(l)/2:]
		for _, i := range lr {
			b.Remove(i)
		}
		return b.Copy().String()
	}
	if err := quick.CheckEqual(f0, f1, nil); err != nil {
		t.Error(err)
	}
}

func TestMax(t *testing.T) {
	f := func(l ascendingInts) bool {
		b := new(Set)
		for _, i := range l {
			b.Add(i)
		}
		// remove last half to test Max's trailing zero logic
		l0, l1 := l[:len(l)/2], l[len(l)/2:]
		for _, i := range l1 {
			b.Remove(i)
		}
		max := b.Max()
		if len(l0) == 0 {
			if max == -1 {
				return true
			}
			t.Logf("b.Max() = %v, expected -1", max)
			return false
		}
		if lMax := l0[len(l0)-1]; max != lMax {
			t.Logf("b.Max() = %v, expected %v", max, lMax)
			return false
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestCount(t *testing.T) {
	f := func(l ascendingInts) bool {
		b := new(Set)
		for _, i := range l {
			b.Add(i)
		}
		if count := b.Cardinality(); count != len(l) {
			t.Logf("b.Count() = %d, expected %d", count, len(l))
			return false
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestNextAfter(t *testing.T) {
	f := func(l ascendingInts) bool {
		b := new(Set)
		for _, i := range l {
			b.Add(i)
		}
		var n int
		var oldi int
		for i := b.NextAfter(-1); i >= 0; i = b.NextAfter(i + 1) {
			if l[n] != i {
				t.Logf("b.NextAfter(%d) = %d, expected %d", oldi, i, l[n])
				return false
			}
			oldi = i
			n++
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestFrom(t *testing.T) {
	f := func(l ascendingInts, fstart float64) bool {
		b := new(Set)
		for _, i := range l {
			b.Add(i)
		}
		start := int(fstart*float64(len(l))) - 1
		got := slices.Collect(b.From(start))
		var want ascendingInts
		for _, num := range l {
			if num >= start {
				want = append(want, num)
			}
		}
		if !slices.Equal(got, want) {
			t.Logf("b.From(%d) = %v, expected %v", start, got, want)
			return false
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestBytes(t *testing.T) {
	f := func(data0 []byte) bool {
		// Get rid of trailing zero bytes
		for len(data0) > 0 && data0[len(data0)-1] == 0 {
			data0 = data0[:len(data0)-1]
		}
		b := new(Set)
		b.FromBytes(data0)
		if data1 := b.Bytes(); bytes.Equal(data0, data1) == false {
			t.Logf("b.Bytes() = %v, expected %v", data1, data0)
			return false
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestString(t *testing.T) {
	f := func(l ascendingInts) bool {
		b := new(Set)
		for _, i := range l {
			b.Add(i)
		}
		if s := b.String(); s != fmt.Sprintf("%v", l) {
			t.Logf("b.String() = %v, wanted %v", s, l)
			return false
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestAddRange(t *testing.T) {
	f0 := func(buf []byte, low, hi uint8) string {
		var s Set
		s.FromBytes(buf)
		for i := int(low); i < int(hi); i++ {
			s.Add(i)
		}
		return s.String()
	}
	f1 := func(buf []byte, low, hi uint8) string {
		var s Set
		s.FromBytes(buf)
		s.AddRange(int(low), int(hi))
		return s.String()
	}
	if err := quick.CheckEqual(f0, f1, nil); err != nil {
		t.Error(err)
	}
}

func TestAddRange00(t *testing.T) {
	b := new(Set)
	b.AddRange(0, 0)
	if b.Cardinality() != 0 {
		t.Errorf("b.String() = %v, expected []", b)
	}
}

func TestRemoveRange00(t *testing.T) {
	b := new(Set)
	b.Add(0)
	b.RemoveRange(0, 0)
	if b.Cardinality() != 1 {
		t.Errorf("b.String() = %v, expected [0]", b)
	}
}

func TestRemoveRange(t *testing.T) {
	f0 := func(buf []byte, low, hi uint8) string {
		var s Set
		s.FromBytes(buf)
		for i := int(low); i < int(hi); i++ {
			s.Remove(i)
		}
		return s.String()
	}
	f1 := func(buf []byte, low, hi uint8) string {
		var s Set
		s.FromBytes(buf)
		s.RemoveRange(int(low), int(hi))
		return s.String()
	}
	if err := quick.CheckEqual(f0, f1, nil); err != nil {
		t.Error(err)
	}
}

func TestEqual(t *testing.T) {
	f0 := func(b0, b1 []byte) bool {
		if len(b0) > 0 && b0[0] >= 127 {
			// give a fair chance of b0 == b1
			b1 = b0
		}
		var s0, s1 Set
		s0.FromBytes(b0)
		s1.FromBytes(b1)
		return bytes.Equal(s0.Bytes(), s1.Bytes())
	}
	f1 := func(b0, b1 []byte) bool {
		if len(b0) > 0 && b0[0] >= 127 {
			// give a fair chance of b0 == b1
			b1 = b0
		}
		var s0, s1 Set
		s0.FromBytes(b0)
		s0.s = append(s0.s, make([]uint, len(s0.s))...)
		s1.FromBytes(b1)
		s1.s = append(s1.s, make([]uint, len(s1.s))...)
		return s0.Equal(&s1)
	}
	if err := quick.CheckEqual(f0, f1, nil); err != nil {
		t.Error(err)
	}
}

func TestBitwise(t *testing.T) {
	for _, v := range []struct {
		op string
		lf func(p, q bool) bool
		bf func(s0, s1 *Set)
	}{
		{
			"intersect",
			func(p, q bool) bool {
				return p && q
			},
			func(s0, s1 *Set) {
				s0.Intersect(s1)
			},
		},
		{
			"subtract",
			func(p, q bool) bool {
				return p && !q
			},
			func(s0, s1 *Set) {
				s0.Subtract(s1)
			},
		},
		{
			"union",
			func(p, q bool) bool {
				return p || q
			},
			func(s0, s1 *Set) {
				s0.Union(s1)
			},
		},
		{
			"symmetric difference",
			func(p, q bool) bool {
				return p != q
			},
			func(s0, s1 *Set) {
				s0.SymmetricDifference(s1)
			},
		},
	} {
		t.Run(v.op, func(t *testing.T) {
			f0 := func(l0, l1 ascendingInts) string {
				b0 := new(Set)
				for _, i := range l0 {
					b0.Add(i)
				}
				b1 := new(Set)
				for _, i := range l1 {
					b1.Add(i)
				}
				v.bf(b0, b1)
				return b0.String()
			}
			f1 := func(l0, l1 ascendingInts) string {
				lx := bitwiseF(v.lf, l0, l1)
				return fmt.Sprint(lx)
			}
			if err := quick.CheckEqual(f0, f1, nil); err != nil {
				t.Error(err)
			}
		})
	}
}

func BenchmarkNextAfter(b *testing.B) {
	buf := make([]byte, 10000)
	rand.Read(buf)
	s := new(Set)
	s.FromBytes(buf)
	var x int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x = s.NextAfter(x)
		if x == -1 {
			x = 0
		}
	}
}

func BenchmarkFrom(b *testing.B) {
	buf := make([]byte, 10000)
	rand.Read(buf)
	s := new(Set)
	s.FromBytes(buf)
	next, stop := iter.Pull(s.From(0))
	defer stop()
	var x int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var ok bool
		x, ok = next()
		if !ok {
			x = 0
		}
	}
	_ = x
}

func bitwiseF(f func(p, q bool) bool, l0, l1 []int) []int {
	var x []int
	lim := max(l0, l1)
	for i := 0; i <= lim; i++ {
		inl0, inl1 := in(i, l0), in(i, l1)
		if f(inl0, inl1) {
			x = append(x, i)
		}
	}
	return x
}

func in(x int, xs []int) bool {
	i := sort.SearchInts(xs, x)
	return i < len(xs) && xs[i] == x
}

func max(a, b []int) int {
	if len(a) == 0 {
		if len(b) == 0 {
			return 0
		}
		return b[len(b)-1]
	}
	if len(b) == 0 {
		return a[len(a)-1]
	}
	ai, bi := a[len(a)-1], b[len(b)-1]
	if ai > bi {
		return ai
	}
	return bi
}

type ascendingInts []int

func (l ascendingInts) Generate(rand *rand.Rand, size int) reflect.Value {
	l = make([]int, rand.Intn(size))
	var x int
	for i := range l {
		x += rand.Intn(bits.UintSize+1) + 1
		l[i] = x
	}
	return reflect.ValueOf(l)
}
