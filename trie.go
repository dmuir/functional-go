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

func GetStats(i IDict) Stats {
	d := i.(dict)
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
	// We don't just return a byte slice since a byte slice is larger than a string.
	// I really wish there was a better way to do this, since we're already creating a lot
	// of work for the GC.
	bytes := []byte(s)
	return string(bytes)
	//return s
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
func expanse(first byte, rest ... byte) expanse_t {
	low := first
	high := first
	for _, v := range rest {
		if v < low { low = v }
		if v > high { high = v }
	}
	return expanse_t{low, high, uint16(high - low) + 1}
}
func (e expanse_t) with(cb byte) expanse_t {
	if cb < e.low { return expanse_t{cb, e.high, uint16(e.high - cb) + 1} }
	if cb > e.high { return expanse_t{e.low, cb, uint16(cb - e.low) + 1} }
	return e
}
func (e expanse_t) contains(cb byte) bool {
	if cb < e.low { return false }
	if cb > e.high { return false }
	return true
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

/*
 create a sequence (represented as a string) of critical bytes.
*/
func critbytes(first byte, rest ... byte) string {
	return string(first) + string(rest)
}

// the max depth is the max number of critical bytes -- the minimum size for depth 32 is 2^32
const maxDepth = 32

func assoc(t itrie, key string, val Value) (itrie, int) {
	var tries [maxDepth]itrie
	var cbs [maxDepth]byte
	path := 0
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
		tries[path] = t
		cbs[path] = cb
		t = t.subAt(cb)
		key = rest
		path++
	}
	// At this point, we have the bottom-most sub trie in r, and tries/cbs has the
	// information about the changes we need to build up the tree
	for path > 0 {
		path--
		t := tries[path]
		cb := cbs[path]
		r = t.with(added, cb, r)
	}
	return r, added
}

func without(t itrie, key string) (itrie, int) {
	if t != nil {
		return t.without(t, key)
	}
	return nil, 0
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
	without(t itrie, key string) (itrie, int)
	withoutValue() (itrie, int)
	count() int
	occupied() int
	expanse() expanse_t
	expanseWithout(byte) expanse_t
	foreach(string, func(string, Value))
	withsubs(start uint, end uint, fn func (byte, itrie))
}

/*
 dict.

 This struct implements the IDict interface via an internal itrie.
*/
type dict struct {
	t itrie
}
func (d dict) Assoc(key string, val Value) IDict {
	t, _ := assoc(d.t, key, val)
	return dict{t}
}
func (d dict) Without(key string) IDict {
	t, _ := without(d.t, key)
	return dict{t}
}
func (d dict) Contains(key string) bool { 
	e := entryAt(d.t, key)
	return e != nil
}
func (d dict) ValueAt(key string) (Value, bool) {
	e := entryAt(d.t, key)
	if e != nil { return e.val(), true }
	return nil, false
}
func (d dict) Count() int {
	if d.t != nil {
		return d.t.count()
	}
	return 0
}
func (d dict) Foreach(fn func(string, Value)) {
	if d.t != nil {
		d.t.foreach("", fn)
	}
}
func (d dict) Iter() chan Item {
	ch := make(chan Item)
	if d.t != nil {
		emit := func(key string, val Value) { ch <- Item{key, val} }
				
		helper := func(t itrie, emit func(string, Value)) {
			t.foreach("", emit)
			close(ch) 
		}
		go helper(d.t, emit)
	} else {
		go close(ch)
	}
	return ch
}

func Dict() IDict {
	return dict{nil}
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

