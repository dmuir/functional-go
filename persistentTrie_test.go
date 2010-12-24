package persistentMap

import (
	"testing"
	"testing/quick"
	"fmt"
	"rand"
	"reflect"
)

func slowcount(bits uint64) int {
	count := 0
	pos := 0
	for bits != 0 {
		bit := uint64(1) << uint(pos)
		if bits & bit != 0 {
			count++
			bits &= ^bit
		}
		pos++
	}
	return count
}

func TestCountbits(t *testing.T) {
	if c := countbits(0x0); c != 0 {
		t.Errorf("TestCountbits: %d != 0", c)
	}
	if c := countbits(0xffffffffffffffff); c != 64 {
		t.Errorf("TestCountbits: %d != 64", c)
	}
	if c := countbits(0x8000000000000000); c != 1 {
		t.Errorf("TestCountbits: %d != 1", c)
	}
	if c := countbits(0x1); c != 1 {
		t.Errorf("TestCountbits: %d != 1", c)
	}

	check := func(x uint64) bool {
		a := countbits(x)
		b := slowcount(x)
		return int(a) == b
	}
	if err := quick.Check(check, nil); err != nil {
		t.Error(err)
	}
}

func BenchmarkCountbits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = countbits(0xfedcba9876543210)
	}
}

func BenchmarkReversebits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = reverse(0xfedcba9876543210)
	}
}

func BenchmarkMinbit(b *testing.B) {
	var bm bitmap_t
	bm.bm[3] = 0x0000000080000000
	for i := 0; i < b.N; i++ {
		_ = bm.min()
	}
}

func BenchmarkMaxbit(b *testing.B) {
	var bm bitmap_t
	bm.bm[0] = 0x0000000080000000
	for i := 0; i < b.N; i++ {
		_ = bm.max()
	}
}

func TestReversebits(t *testing.T) {
	if c := reverse(0x0); c != 0 {
		t.Errorf("TestReversebits: %x != 0", c)
	}
	if c := reverse(0xffffffffffffffff); c != 0xffffffffffffffff {
		t.Errorf("TestReversebits: %x != 0xffffffffffffffff", c)
	}
	if c := reverse(0x1); c != 0x8000000000000000 {
		t.Errorf("TestReversebits: %x != 0x8000000000000000", c)
	}
	if c := reverse(0x8000000000000000); c != 0x1 {
		t.Errorf("TestReversebits: %x != 0x1", c)
	}
	if c := reverse(0x0f0f000000000000); c != 0x000000000000f0f0 {
		t.Errorf("TestReversebits: %x != 0x000000000000f0f0", c)
	}
}

func TestMinbit(t *testing.T) {
	var b bitmap_t
	b.bm[0] = 0x1
	if c := b.min(); c != 0 {
		t.Errorf("TestMinbit: %d != 0", c)
	}
	b.bm[0] = 1 << 55
	if c := b.min(); c != 55 {
		t.Errorf("TestMinbit: %d != 55", c)
	}
	b.bm[0] |= 0xff00000000000000
	if c := b.min(); c != 55 {
		t.Errorf("TestMinbit: %d != 55", c)
	}
	b.bm[0] = 0
	b.bm[2] = 0x1
	if c := b.min(); c != 128 {
		t.Errorf("TestMinbit: %d != 128", c)
	}
}

func TestMaxbit(t *testing.T) {
	var b bitmap_t
	b.bm[0] = 0x1
	if c:= b.max(); c != 0 {
		t.Errorf("TestMaxbit: %d != 0", c)
	}
	b.bm[0] = 1 << 55
	if c := b.max(); c != 55 {
		t.Errorf("TestMaxBit: %d != 55", c)
	}
	b.bm[0] |= 0x007fffffffffffff
	if c := b.max(); c != 55 {
		t.Errorf("TestMaxBit: %d != 55", c)
	}
	b.bm[2] = 0x02
	if c := b.max(); c != 129 {
		t.Errorf("TestMaxbit: %d != 129", c)
	}
}
	
func TestEmptyTrie(t *testing.T) {
	m := NewTrie()
	if m.Count() != 0 {
		t.Errorf("TestEmptyMap: Count (%d) != 0", m.Count())
	}
}

func TestAssocTrie(t *testing.T) {
	m0 := NewTrie()

	m1 := m0.Assoc("foo", "bar")
	if m0.Count() != 0 {
		t.Errorf("TestAssoc: m0.Count (%d) != 0", m0.Count())
	}
	if m1.Count() != 1 {
		t.Errorf("TestAssoc: m1.Count (%d) != 1", m1.Count())
	}
		
	m2 := m1.Assoc("bar", "baz")
	if m0.Count() != 0 {
		t.Errorf("TestAssoc: m0.Count (%d) != 0", m0.Count())
	}
	if m1.Count() != 1 {
		t.Errorf("TestAssoc: m1.Count (%d) != 1", m1.Count())
	}
	if m2.Count() != 2 {
		t.Errorf("TestAssoc: m2.Count (%d) != 2", m2.Count())
	}

	if m1.ValueAt("foo").(string) != "bar" {
		t.Errorf("TestAssoc: m1.ValueAt('foo') (%s) != 'bar'",
			m1.ValueAt("foo").(string))
	}
	m1 = m1.Assoc("foo", "blah")
	if m1.ValueAt("foo").(string) != "blah" {
		t.Errorf("TestAssoc: m1.ValueAt('foo') (%s) != 'blah'",
			m1.ValueAt("foo").(string))
	}
	if m2.ValueAt("foo").(string) != "bar" {
		t.Errorf("TestAssoc: m2.ValueAt('foo') (%s) != 'bar'",
			m2.ValueAt("foo").(string))
	}
}

func printItems(m IPersistentMap) {
	typ := reflect.Typeof(m)
	fmt.Printf("Dumping map(type=%s)...\n", typ.String())
	for item := range m.Iter() {
		fmt.Printf("%s: %d\n", item.key, item.val.(int))
	}
}

func TestWithoutTrie(t *testing.T) {
	m := NewTrie()
	m = m.Assoc("A", 14)
	m = m.Assoc("K", 13)
	m = m.Assoc("Q", 12)
	m = m.Assoc("J", 11)
	m = m.Assoc("T", 10)
	m = m.Assoc("9", 9)
	m = m.Assoc("8", 8)
	m = m.Assoc("7", 7)
	m = m.Assoc("6", 6)
	m = m.Assoc("5", 5)
	m = m.Assoc("4", 4)
	m = m.Assoc("3", 3)
	m = m.Assoc("2", 2)

	if !m.Contains("T") {
		t.Errorf("TestWithout: m.Contains('T') is false.")
	}
	w := m.Without("T")
	if w.Contains("T") {
		t.Errorf("TestWithout: w.Contains('T') is true.")
	}
	if !m.Contains("T") {
		t.Errorf("TestWithout: m.Contains('T') is false.")
	}
	w = w.Without("J")
	if w.Contains("J") {
		t.Errorf("TestWithout: w.Contains('J') is true.")
	}
	w = w.Without("Q")
	if w.Contains("Q") {
		t.Errorf("TestWithout: w.Contains('Q') is true.")
	}
	w = w.Without("K")
	if w.Contains("K") {
		t.Errorf("TestWithout: w.Contains('K') is true.")
	}
	w = w.Without("A")
	if w.Contains("A") {
		t.Errorf("TestWithout: w.Contains('A') is true.")
	}
	if w.Count() != 8 {
		t.Errorf("TestWithout: w.Count() != 8 (%d)", w.Count())
	}
}

func TestIterTrie(t *testing.T) {
	var keys [256]string
	m := NewTrie()

	for i := 0; i < 256; i++ {
		keys[i] = fmt.Sprintf("%02x", 255 - i)
		m = m.Assoc(keys[i], i)
	}

	if m.Count() != 256 {
		t.Errorf("TestIter: m.Count() (%d) != 256", m.Count())
	}

	count := 0
	for item := range m.Iter() {
		count++
		i := item.val.(int)
		if item.key != keys[i] {
			t.Errorf("TestIter: (%d) %s != %s", i, item.key, keys[i])
		}
	}
	if count != m.Count() {
		t.Errorf("TestIter: only iterated %d of %d items.", count, m.Count())
	}
}
		
func randomKey() string {
	return fmt.Sprintf("%x", rand.Int63())
}

func BenchmarkKeys(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = randomKey()
	}
}

func BenchmarkAssoc(b *testing.B) {
	m := NewTrie()
	for i := 0; i < b.N; i++ {
		m = m.Assoc(randomKey(), i)
	}
}