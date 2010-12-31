package immutable

import (
	"testing"
	"testing/quick"
	"fmt"
	"os"
	"rand"
	"reflect"
	"runtime"
	"runtime/pprof"
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
	var bm bitmap_
	bm.bm[3] = 0x0000000080000000
	for i := 0; i < b.N; i++ {
		_ = bm.min()
	}
}

func BenchmarkMaxbit(b *testing.B) {
	var bm bitmap_
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
	var b bitmap_
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
	var b bitmap_
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

func checkExpanse(e, ex expanse_t, t *testing.T) {
	if e.low > e.high {
		t.Errorf("e.low must be <= e.high (%d(%c) > %d(%c))", e.low, e.low, e.high, e.high)
	}
	if e.low != ex.low {
		t.Errorf("Expected e.low: %d(%c), got %d(%c)", ex.low, ex.low, e.low, e.low)
	}
	if e.high != ex.high {
		t.Errorf("Expected e.high: %d(%c), got %d(%c)", ex.high, ex.high, e.high, e.high)
	}
	if e.size != ex.size {
		t.Errorf("Expected e.size: %d, got %d", ex.size, e.size)
	}
}

func TestExpanse(t *testing.T) {
	checkExpanse(expanse('a', 'b'), expanse_t{'a', 'b', 2}, t)
	checkExpanse(expanse('b', 'a'), expanse_t{'a', 'b', 2}, t)
	checkExpanse(expanse(0, 255), expanse_t{0, 255, 256}, t)
	checkExpanse(expanse(255, 0), expanse_t{0, 255, 256}, t)
	checkExpanse(expanse(1, 10).with(9), expanse_t{1, 10, 10}, t)
	checkExpanse(expanse(10, 1).with(11), expanse_t{1, 11, 11}, t)
}

func checkTrie(b itrie, exoccupied int, exp expanse_t, check int, t *testing.T) {
	if t != nil {
		if occupied := b.occupied(); occupied != exoccupied {
			t.Errorf("Expected occupied = %d, got %d", exoccupied, occupied)
		}
		checkExpanse(b.expanse(), exp, t)
	}
}
		
func testBag(t *testing.T) itrie {
	check := 1
	b := bag2("", nil, false, 'f', 'b', leaf("oo", 1), leaf("ar", 2))
	checkTrie(b, 2, expanse('f', 'b'), check, t); check++
	b = bag(b, 'e', leaf("at", 4))
	checkTrie(b, 3, expanse('f', 'b'), check, t); check++
	b = bag(b, 'a', leaf("te", 5))
	checkTrie(b, 4, expanse('f', 'a'), check, t); check++
	b = bag(b, 'd', leaf("og", 7))
	checkTrie(b, 5, expanse('f', 'a'), check, t); check++
	return b
}

func TestBag(t *testing.T) {
	b := testBag(t)
	count := 0
	b.withsubs(0, 256, func(cb byte, t itrie) { count++ })
	if count != 5 {
		t.Errorf("Expected 5 sub-tries, got %d", count)
	}
	e1 := b.expanseWithout('a')
	b1 := bagWithout(b, e1, 'a')
	checkTrie(b1, 4, expanse('f', 'b'), 1, t)
	e2 := b.expanseWithout('e')
	b2 := bagWithout(b, e2, 'e')
	checkTrie(b2, 4, expanse('f', 'a'), 2, t)
	e3 := b1.expanseWithout('f')
	b3 := bagWithout(b1, e3, 'f')
	checkTrie(b3, 3, expanse('e', 'b'), 3, t)
}
func TestBagWith(t *testing.T) {
	b := bag2("", nil, false, '0', '1', leaf("00", 1), leaf("00", 2))
	b1 := b.with(1, '2', leaf("00", 3))
	if b.count() != 2 { t.Errorf("Expected b.count() == 2, got %d", b.count()) }
	if b1.count() != 3 { t.Errorf("Expected b1.count() == 3, got %d", b1.count()) }
	if e := entryAt(b, "000"); e == nil { t.Error("Expected entry @ '000'")
	} else {
		if !e.hasVal() { t.Error("Expected entry @ '000' to have a value") }
		if v, ok := e.val().(int); !ok || v != 1 {
			t.Errorf("Expected v == 1, got %d", v) 
		}
	}
	if e := entryAt(b, "200"); e != nil { t.Error("Expected no entry @ '200'") }
	if e := entryAt(b1, "200"); e == nil { t.Error("Expected entry @ '200'")
	} else {
		if !e.hasVal() { t.Error("Expected entry @ '200' to have a value") }
		if v, ok := e.val().(int); !ok || v != 3 {
			t.Errorf("Expected v == 3, got %d", v) 
		}
	}
	b2 := b1.with(0, '2', leaf("00", 4))
	if e := entryAt(b2, "200"); e == nil { t.Error("Expected entry @ '200'")
	} else {
		if !e.hasVal() { t.Error("Expected entry @ '200' to have a value") }
		if v, ok := e.val().(int); !ok || v != 4 {
			t.Errorf("Expected v == 4, got %d", v) 
		}
	}
}

func TestSpan(t *testing.T) {
	check := 1
	b := testBag(nil)
	e := b.expanse()
	e = e.with('c')
	s := span(b, e, 'c', leaf("ar", 8))
	checkTrie(s, 6, expanse('a', 'f'), check, t); check++
	e = e.with('g')
	s = span(s, e, 'g', leaf("irl", 9))
	checkTrie(s, 7, expanse('a', 'g'), check, t); check++
	
	e1 := s.expanseWithout('c')
	s1 := spanWithout(s, e1, 'c')
	checkTrie(s1, 6, expanse('a', 'g'), check, t); check++
	e2 := s.expanseWithout('a')
	s2 := spanWithout(s, e2, 'a')
	checkTrie(s2, 6, expanse('b', 'g'), check, t); check++
	e3 := s2.expanseWithout('g')
	s3 := spanWithout(s2, e3, 'g')
	checkTrie(s3, 5, expanse('b', 'f'), check, t); check++
}

func TestSpanWith(t *testing.T) {
	b := bag2("", nil, false, '0', '1', leaf("00", 1), leaf("00", 2))
	s := span(b, b.expanse().with('2'), '2', leaf("00", 3))
	s1 := s.with(1, '3', leaf("00", 5))
	if s.count() != 3 { t.Errorf("Expected s.count() == 3, got %d", s.count()) }
	if s1.count() != 4 { t.Errorf("Expected s1.count() == 4, got %d", s1.count()) }
	if e := entryAt(s, "000"); e == nil { t.Error("Expected entry @ '000'")
	} else {
		if !e.hasVal() { t.Error("Expected entry @ '000' to have a value") }
		if v, ok := e.val().(int); !ok || v != 1 {
			t.Errorf("Expected v == 1, got %d", v) 
		}
	}
	if e := entryAt(s, "300"); e != nil { t.Error("Expected no entry @ '300'") }
	if e := entryAt(s1, "300"); e == nil { t.Error("Expected entry @ '300'")
	} else {
		if !e.hasVal() { t.Error("Expected entry @ '300' to have a value") }
		if v, ok := e.val().(int); !ok || v != 5 {
			t.Errorf("Expected v == 3, got %d", v) 
		}
	}
	s2 := s1.with(0, '2', leaf("00", 4))
	if e := entryAt(s2, "200"); e == nil { t.Error("Expected entry @ '200'")
	} else {
		if !e.hasVal() { t.Error("Expected entry @ '200' to have a value") }
		if v, ok := e.val().(int); !ok || v != 4 {
			t.Errorf("Expected v == 4, got %d", v) 
		}
	}
}

func printExpanse(e expanse_t) {
	fmt.Printf("e.low = %d(%c), e.high = %d(%c), e.size = %d\n", e.low, e.low, e.high, e.high, e.size)
}
func printBitmap(b [4]uint64) {
	for _, bm := range b {
		fmt.Printf("               %064b\n", bm)
	}
}

func TestBitmap(t *testing.T) {
	check := 1
	b := testBag(nil)
	bm := bitmap(b, 'c', leaf("ar", 8))
	checkTrie(bm, 6, expanse('a', 'f'), check, t); check++
	bm = bitmap(bm, 'g', leaf("irl", 9))
	checkTrie(bm, 7, expanse('a', 'g'), check, t); check++
	
	e1 := bm.expanseWithout('c')
	bm1 := bitmapWithout(bm, e1, 'c')
	checkTrie(bm1, 6, expanse('a', 'g'), check, t); check++
	e2 := bm.expanseWithout('a')
	bm2 := bitmapWithout(bm, e2, 'a')
	checkTrie(bm2, 6, expanse('b', 'g'), check, t); check++
	e3 := bm2.expanseWithout('g')
	bm3 := bitmapWithout(bm2, e3, 'g')
	checkTrie(bm3, 5, expanse('b', 'f'), check, t); check++
}
	
func TestBitmapWith(t *testing.T) {
	b := bag2("", nil, false, '0', '1', leaf("00", 1), leaf("00", 2))
	bm := bitmap(b, '2', leaf("00", 3))
	bm1 := bm.with(1, '3', leaf("00", 5))
	if bm.count() != 3 { t.Errorf("Expected bm.count() == 3, got %d", bm.count()) }
	if bm1.count() != 4 { t.Errorf("Expected bm1.count() == 4, got %d", bm1.count()) }
	if e := entryAt(bm, "000"); e == nil { t.Error("Expected entry @ '000'")
	} else {
		if !e.hasVal() { t.Error("Expected entry @ '000' to have a value") }
		if v, ok := e.val().(int); !ok || v != 1 {
			t.Errorf("Expected v == 1, got %d", v) 
		}
	}
	if e := entryAt(bm, "300"); e != nil { t.Error("Expected no entry @ '300'") }
	if e := entryAt(bm1, "300"); e == nil { t.Error("Expected entry @ '300'")
	} else {
		if !e.hasVal() { t.Error("Expected entry @ '300' to have a value") }
		if v, ok := e.val().(int); !ok || v != 5 {
			t.Errorf("Expected v == 3, got %d", v) 
		}
	}
	bm2 := bm1.with(0, '2', leaf("00", 4))
	if e := entryAt(bm2, "200"); e == nil { t.Error("Expected entry @ '200'")
	} else {
		if !e.hasVal() { t.Error("Expected entry @ '200' to have a value") }
		if v, ok := e.val().(int); !ok || v != 4 {
			t.Errorf("Expected v == 4, got %d", v) 
		}
	}
}

func TestEmptyTrie(t *testing.T) {
	m := Dict()
	if m.Count() != 0 {
		t.Errorf("TestEmptyMap: Count (%d) != 0", m.Count())
	}
}

func TestAssocTrie(t *testing.T) {
	m0 := Dict()

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

	if v, _ := m1.ValueAt("foo"); v.(string) != "bar" {
		t.Errorf("TestAssoc: m1.ValueAt('foo') (%s) != 'bar'", v.(string))
	}
	m1 = m1.Assoc("foo", "blah")
	if v, _ := m1.ValueAt("foo"); v.(string) != "blah" {
		t.Errorf("TestAssoc: m1.ValueAt('foo') (%s) != 'blah'", v.(string))
	}
	if v, _ := m2.ValueAt("foo"); v.(string) != "bar" {
		t.Errorf("TestAssoc: m2.ValueAt('foo') (%s) != 'bar'", v.(string))
	}
}

func printItems(m IDict) {
	typ := reflect.Typeof(m)
	fmt.Printf("Dumping map(type=%s)...\n", typ.String())
	for item := range m.Iter() {
		fmt.Printf("%s: %d\n", item.key, item.val.(int))
	}
}

func TestWithoutTrie(t *testing.T) {
	m := Dict()
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
	m := Dict()

	for i := 0; i < 256; i++ {
		keys[i] = fmt.Sprintf("%02x", 255 - i)
		m = m.Assoc(keys[i], i)
	}

	if m.Count() != 256 {
		t.Errorf("TestIter: m.Count() (%d) != 256", m.Count())
	}

	for i, k := range keys {
		if !m.Contains(k) {
			t.Errorf("TestIter: m doesn't contain %s", k)
		} else {
			val, ok := m.ValueAt(k)
			if !ok || val.(int) != i {
				t.Errorf("TestIter: m@%s(%d) != %d", k, val.(int), i)
			}
		}
	}

	count := 0
	m.Foreach(func(key string, val Value) {
		i := val.(int)
		if i != 255 - count {
			fmt.Printf("skipped...\n")
			count = 255 - val.(int)
		}
		count++
		if key != keys[i] {
			t.Errorf("TestIter: (%d) %s != %s", i, key, keys[i])
		}
	})
	if count != m.Count() {
		t.Errorf("TestIter: only iterated %d of %d items via Foreach.", count, m.Count())
	}

	count = 0
	for item := range m.Iter() {
		if item.val.(int) != 255 - count {
			fmt.Printf("skipped...\n")
			count = 255 - item.val.(int)
		}
		count++
		i := item.val.(int)
		if item.key != keys[i] {
			t.Errorf("TestIter: (%d) %s != %s", i, item.key, keys[i])
		}
	}
	if count != m.Count() {
		t.Errorf("TestIter: only iterated %d of %d items via Iter.", count, m.Count())
	}
}
func randomKey() string {
	r := rand.Int63n(1000000000) + 10000000000
	return fmt.Sprintf("%x", r)
}

var alloc int64
var total int64
var lookups int64
var mallocs int64
var numGC int32
var pauseNs int64

func snapshotGC() {
	alloc = -int64(runtime.MemStats.Alloc)
	total = -int64(runtime.MemStats.TotalAlloc)
	lookups = -int64(runtime.MemStats.Lookups)
	mallocs = -int64(runtime.MemStats.Mallocs)
	numGC = -int32(runtime.MemStats.NumGC)
	pauseNs = -int64(runtime.MemStats.PauseNs)
}

func printGC() {
	alloc += int64(runtime.MemStats.Alloc)
	total += int64(runtime.MemStats.TotalAlloc)
	lookups += int64(runtime.MemStats.Lookups)
	mallocs += int64(runtime.MemStats.Mallocs)
	numGC += int32(runtime.MemStats.NumGC)
	pauseNs += int64(runtime.MemStats.PauseNs)

	fmt.Printf("alloc: %d\ntotal: %d\n", alloc, total)
	fmt.Printf("lookups: %d\nmallocs: %d\n", lookups, mallocs)
	fmt.Printf("numGC: %d\npauseNs: %d\navg. pauseNs:%d\n", numGC, pauseNs, pauseNs/int64(numGC))
}	
	
func TestRandomAssoc(t *testing.T) {
	d := Dict()
	m := map[string]int{}

	for i := 0; i < 100000; i++ {
		key := randomKey()
		d = d.Assoc(key, i)
		m[key] = i
	}
	if d.Count() != len(m) {
		t.Errorf("Expected Count == %d, got %d", len(m), d.Count())
	}

	for k, v := range m {
		if !d.Contains(k) {
			t.Errorf("%s not in dict.", k)
			continue
		}
		if vt, ok := d.ValueAt(k); !ok || vt.(int) != v {
			t.Errorf("Expected %d at %s, got %d", v, k, vt)
		}
	}
}

func TestPrintSizes(t *testing.T) {
	fmt.Printf("Sizes of internal structures.\n")
	fmt.Printf("sizeof(leafV): %d\n", reflect.Typeof(leafV{}).Size())
	fmt.Printf("sizeof(leafKV): %d\n", reflect.Typeof(leafKV{}).Size())
	fmt.Printf("sizeof(bag_): %d\n", reflect.Typeof(bag_{}).Size())
	fmt.Printf("sizeof(bagK): %d\n", reflect.Typeof(bagK{}).Size())
	fmt.Printf("sizeof(bagV): %d\n", reflect.Typeof(bagV{}).Size())
	fmt.Printf("sizeof(bagKV): %d\n", reflect.Typeof(bagKV{}).Size())
	fmt.Printf("sizeof(span_): %d\n", reflect.Typeof(span_{}).Size())
	fmt.Printf("sizeof(spanK): %d\n", reflect.Typeof(spanK{}).Size())
	fmt.Printf("sizeof(spanV): %d\n", reflect.Typeof(spanV{}).Size())
	fmt.Printf("sizeof(spanKV): %d\n", reflect.Typeof(spanKV{}).Size())
	fmt.Printf("sizeof(bitmap_): %d\n", reflect.Typeof(bitmap_{}).Size())
	fmt.Printf("sizeof(bitmapK): %d\n", reflect.Typeof(bitmapK{}).Size())
	fmt.Printf("sizeof(bitmapV): %d\n", reflect.Typeof(bitmapV{}).Size())
	fmt.Printf("sizeof(bitmapKV): %d\n", reflect.Typeof(bitmapKV{}).Size())
}
func TestRandomAssocStats(t *testing.T) {
	const num = 100000
	keys := make([]string, num)
	values :=  make([]Value, num)
	for i, _ := range keys {
		keys[i] = randomKey()
		values[i] = i
	}
	ResetCumulativeStats()
	runtime.GC()
	snapshotGC()

	d := Dict()

	for i, key := range keys {
		d = d.Assoc(key, values[i])
	}

	fmt.Printf("Information for Dict...\n")
	PrintStats(GetStats(d))
	fmt.Printf("Cumulative Information for Dicts...\n")
	PrintStats(Cumulative)
	runtime.GC()
	fmt.Println("Memory Info...")
	printGC()

	fmt.Println("And one more...")
	key := randomKey()
	var val Value = "foobar" 
	ResetCumulativeStats()
	runtime.GC()
	snapshotGC()
	d = d.Assoc(key, val)
	f, err := os.Open("mem-pre-gc.pprof", os.O_WRONLY|os.O_CREAT, 0777)
	if err == nil {
		pprof.WriteCompleteHeapProfile(f)
	} else {
		fmt.Println(err.String())
	}
	runtime.GC()
	fmt.Printf("Incremental Cumulative Stats...\n")
	PrintStats(Cumulative)
	fmt.Println("Memory Info...")
	printGC()
}

func BenchmarkAssoc(b *testing.B) {
	b.StopTimer()
	keys := make([]string, b.N)
	values :=  make([]Value, b.N)
	for i, _ := range keys {
		keys[i] = randomKey()
		values[i] = i
	}
	runtime.GC()
	b.StartTimer()
	d := Dict()
	for i, key := range keys {
		d = d.Assoc(key, values[i])
	}
}
