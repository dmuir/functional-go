package immutable

import (
	"fmt"
	"reflect"
)

const (
	kLeafV = iota
	kLeafKV
	kBag_
	kBagK
	kBagV
	kBagKV
	kSpan_
	kSpanK
	kSpanV
	kSpanKV
	kBitmap_
	kBitmapK
	kBitmapV
	kBitmapKV
	
	numVariants
)

type Stats [numVariants]int
var Cumulative Stats

func GetStats(d Dict) Stats {
	var stats Stats
	if d.t != nil {
		var collect func(byte, itrie)
		collect = func(cb byte, t itrie) {	
			switch n := t.(type) {
			case *leafV: stats[kLeafV]++
			case *leafKV: stats[kLeafKV]++
			case *bag_: stats[kBag_]++
			case *bagK: stats[kBagK]++
			case *bagV: stats[kBagV]++
			case *bagKV: stats[kBagKV]++
			case *span_: stats[kSpan_]++
			case *spanK: stats[kSpanK]++
			case *spanV: stats[kSpanV]++
			case *spanKV: stats[kSpanKV]++
			case *bitmap_: stats[kBitmap_]++
			case *bitmapK: stats[kBitmapK]++
			case *bitmapV: stats[kBitmapV]++
			case *bitmapKV: stats[kBitmapKV]++
			}
			t.withsubs(0, 256, collect)
		}
		collect(0, d.t)
	}
	return stats
}

func ResetCumulativeStats() {
	for i, _ := range Cumulative {
		Cumulative[i] = 0
	}
}

func PrintStats(stats Stats) {
	statNames := [numVariants]string{
		"leafV",
		"leafKV",
		"bag_",
		"bagK",
		"bagV",
		"bagKV",
		"span_",
		"spanK",
		"spanV",
		"spanKV",
		"bitmap_",
		"bitmapK",
		"bitmapV",
		"bitmapKV",
	}
	sizes := [numVariants]uintptr{
		reflect.Typeof(leafV{}).Size(),
		reflect.Typeof(leafKV{}).Size(),
		reflect.Typeof(bag_{}).Size(),
		reflect.Typeof(bagK{}).Size(),
		reflect.Typeof(bagV{}).Size(),
		reflect.Typeof(bagKV{}).Size(),
		reflect.Typeof(span_{}).Size(),
		reflect.Typeof(spanK{}).Size(),
		reflect.Typeof(spanV{}).Size(),
		reflect.Typeof(spanKV{}).Size(),
		reflect.Typeof(bitmap_{}).Size(),
		reflect.Typeof(bitmapK{}).Size(),
		reflect.Typeof(bitmapV{}).Size(),
		reflect.Typeof(bitmapKV{}).Size(),
	}
	for i, v := range stats {
		fmt.Printf("%s: %d (%d)\n", statNames[i], v, uintptr(v)*sizes[i])
	}
}	
const maxBagSize = 7
const minSpanSize = 4
const maxSpanWaste = 4

func str(s string) string {
	// We do this to ensure that the string is a new copy and not a slice of a larger string
	bytes := []byte(s)
	return string(bytes)
}

func abs(x int) int {
	if x < 0 { return -x }
	return x
}

func min(a, b int) int {
	if a < b { return a }
	return b
}

func max(a, b int) int {
	if a > b { return a }
	return b
}

type expanse_t struct {
	low byte
	high byte
	size uint16
}
func expanse0() expanse_t { return expanse_t{0, 0, 0} }
func expanse(low byte, high byte) expanse_t {
	if low > high { low, high = high, low }
	return expanse_t{low, high, uint16(high - low) + 1}
}
func (e expanse_t) with(cb byte) expanse_t {
	if cb < e.low { return expanse_t{cb, e.high, uint16(e.high - cb) + 1} }
	if cb > e.high { return expanse_t{e.low, cb, uint16(cb - e.low) + 1} }
	return e
}

func spanOK(e expanse_t, count int) bool {
	return int(e.size) <= (count + maxSpanWaste)
}

/*
 finds the location of the critical byte for 2 strings -- this is the first byte at which the
 strings differ.  The second return value indicates whether the strings match exactly.
*/
func findcb(a string, b string) (int, bool) {
	la := len(a)
	lb := len(b)
	l := min(la, lb)
	i := 0
	for ; i < l; i++ {
		if a[i] != b[i] {
			return i, false
		}
	}
	return i, la == lb
}

/*
 split a string at the critical byte, returning the portion of the string preceeding the critical
 byte, the critical byte, and the portion after the critical byte.
*/
func splitKey(key string, crit int) (string, byte, string) {
	if crit >= len(key) {
		return key, 0, ""
	}
	return key[0:crit], key[crit], key[crit+1:]
}

const segSize = 32
type trieStack struct {
	t [segSize]itrie
	cb [segSize]byte
	pos int
	next *trieStack
}
func (s *trieStack) reset() *trieStack {
	s.next = nil
	s.pos = 0
	return s
}
func (s *trieStack) push(cb byte, t itrie) *trieStack {
	if s.pos >= segSize {
		fmt.Println("Growing stack")
		s.pos--
		next := new(trieStack)
		next.next = s
		s = next
	}
	s.t[s.pos] = t
	s.cb[s.pos] = cb
	s.pos++
	return s
}
func (s *trieStack) pop() (itrie, byte, *trieStack, bool) {
	s.pos--
	if s.pos < 0 {
		s = s.next
	}
	if s == nil { return nil, 0, nil, true }
	return s.t[s.pos], s.cb[s.pos], s, false
}

// Use a global variable to avoid putting the first stack segment on the heap
var stack trieStack

func assoc(t itrie, key string, val Value) (itrie, int) {
	s := stack.reset()
	var r itrie
	var added int

	for {
		if t == nil {
			r, added = leaf(key, val), 1
			break
		}
		key_ := t.key()
		crit, match := findcb(key, key_)
		if match {
			r, added = t.cloneWithKeyValue(key, val)
			break
		}
		
		prefix, cb, rest := splitKey(key, crit)
		_, cb_, rest_ := splitKey(key_, crit)

		if crit < len(key_) {
			added = 1
			if crit == len(key) {
				r = bag1(prefix, val, true, cb_, t.cloneWithKey(rest_))
			} else {
				r = bag2(prefix, nil, false, cb, cb_,
					leaf(rest, val), t.cloneWithKey(rest_))
			}
			break
		}
		s = s.push(cb, t)
		t = t.subAt(cb)
		key = rest
	}
	// At this point, we have the bottom-most sub trie in r, and tries/cbs has the
	// information about the changes we need to build up the tree
	for s != nil {		
		t, cb, next, done := s.pop()
		if done { break }
		r = t.with(added, cb, r)
		s = next
	}
	return r, added
}

func without(t itrie, key string) (itrie, int) {
	s := stack.reset()
	r := t
	removed := 0

	for t != nil {
		key_ := t.key()
		crit, match := findcb(key, key_)
		if crit < len(key_) {
			// we don't have the element being removed
			return r, 0
		}
		if match {
			if !t.hasVal() {
				// don't have the element being removed
				return r, 0
			}
			r, removed = t.withoutValue()
			break
		}
		if crit >= len(key) {
			// we don't have the element being removed
			return r, 0
		}

		_, cb, rest := splitKey(key, crit)

		s = s.push(cb, t)
		t = t.subAt(cb)
		key = rest
	}
	// At this point, we have the bottom most sub trie (possibly nil) in r, and tries/cbs
	// has the information about the changes we need to build up the tree
	for s != nil {
		t, cb, next, done := s.pop()
		if done { break }
		r = t.without(cb, r)
		s = next
	}
	return r, removed
}

func entryAt(t itrie, key string) itrie {
	for t != nil {
		crit, match := findcb(key, t.key())
		if match && t.hasVal() { return t }
		if crit >= len(key) { return nil }
		_, cb, rest := splitKey(key, crit)
		t = t.subAt(cb)
		key = rest
	}
	return t
}

/*
 itrie.

 This is the interface of the internal trie nodes.
*/
type itrie interface {
	key() string
	hasVal() bool
	val() Value
	subAt(cb byte) itrie
	with(incr int, cb byte, r itrie) itrie
	modify(incr, i int, t itrie) itrie
	cloneWithKey(string) itrie
	cloneWithKeyValue(string, Value) (itrie, int)
	without(cb byte, r itrie) itrie
	withoutValue() (itrie, int)
	count() int
	occupied() int
	expanse() expanse_t
	expanseWithout(byte) expanse_t
	foreach(string, func(string, Value))
	withsubs(start uint, end uint, fn func (byte, itrie))
}

/*
 Functions which capture common behavior.
*/
type ientry interface {
	key() string
	val() Value
	hasVal() bool
}

type entry_ struct {}
func (e entry_) key() string { return "" }
func (e entry_) val() Value { return nil }
func (e entry_) hasVal() bool { return false }

type entryK struct {
	key_ string
}
func (e entryK) key() string { return e.key_ }
func (e entryK) val() Value { return nil }
func (e entryK) hasVal() bool { return false }

type entryV struct {
	val_ Value
}
func (e entryV) key() string { return "" }
func (e entryV) val() Value { return e.val_ }
func (e entryV) hasVal() bool { return true }

type entryKV struct {
	key_ string
	val_ Value
}
func (e entryKV) key() string { return e.key_ }
func (e entryKV) val() Value { return e.val_ }
func (e entryKV) hasVal() bool { return true }

