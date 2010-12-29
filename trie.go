package immutable

import (
	"fmt"
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
			case *bitmap_t: 
				if len(n.key_) > 0 {
					if n.full {
						stats[kBitmapKV]++
					} else {
						stats[kBitmapK]++
					}
				} else {
					if n.full {
						stats[kBitmapV]++
					} else {
						stats[kBitmap_]++
					}
				}
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

	for i, v := range stats {
		fmt.Printf("%s: %d\n", statNames[i], v)
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


func assoc(t itrie, key string, val Value) (itrie, int) {
	if t != nil {
		crit, match := findcb(key, t.key())
		if match {
			return t.cloneWithKeyValue(key, val)
		}
		
		prefix, cb, rest := splitKey(key, crit)
		_, _cb, _rest := splitKey(t.key(), crit)

		if crit < len(t.key()) {
			if crit == len(key) {
				return bag1(prefix, val, true, _cb, t.cloneWithKey(_rest)), 1
			}
			return bag2(prefix, nil, false, cb, _cb,
				leaf(rest, val), t.cloneWithKey(_rest)), 1
		}
		return t.assoc(t, prefix, cb, rest, val)
	}
	return leaf(key, val), 1
}

func without(t itrie, key string) (itrie, int) {
	if t != nil {
		return t.without(t, key)
	}
	return nil, 0
}

/*
 itrie.

 This is the interface of the internal trie nodes.
*/
type itrie interface {
	key() string
	hasVal() bool
	val() Value
	modify(incr, i int, t itrie) itrie
	cloneWithKey(string) itrie
	cloneWithKeyValue(string, Value) (itrie, int)
	assoc(t itrie, prefix string, cb byte, rest string, val Value) (itrie, int)
	without(t itrie, key string) (itrie, int)
	withoutValue() (itrie, int)
	entryAt(string) itrie
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
	if d.t != nil {
		e := d.t.entryAt(key)
		return e != nil
	}
	return false
}
func (d dict) ValueAt(key string) Value {
	if d.t != nil {
		e := d.t.entryAt(key)
		if e != nil { return e.val() }
	}
	panic(fmt.Sprintf("no value at: %s", key))
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

/*
 leaf_t

 Internal node that contains only a key and a value.
*/
type leafV struct {
	entryV
}
type leafKV struct {
	key_ string
	leafV
}

func leaf(key string, val Value) itrie {
	if len(key) > 0 {
		Cumulative[kLeafKV]++
		l := new(leafKV)
		l.key_ = key; l.val_ = val
		return l
	}
	Cumulative[kLeafV]++
	l := new(leafV)
	l.val_ = val
	return l
}
func (l *leafV) modify(incr, i int, sub itrie) itrie {
	panic("can't modify a leaf in this way")
}
func (l *leafV) cloneWithKey(key string) itrie {
	return leaf(key, l.val_)
}
func (l *leafV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	return leaf(key, val), 0
}
func (l *leafV) withoutValue() (itrie, int) {
	panic("Can't have a leaf without a value")
}
func (l *leafV) assoc(t itrie, prefix string, cb byte, rest string, val Value) (itrie, int) {
	return bag1(prefix, l.val(), true, cb, leaf(rest, val)), 1
}
func (l *leafV) without(t itrie, key string) (itrie, int) {
	if len(key) == 0 { return nil, 1 }
	return t, 0
}
func (l *leafKV) without(t itrie, key string) (itrie, int) {
	if key == l.key_ { return nil, 1 }
	return t, 0
}
func (l *leafV) entryAt(key string) itrie {
	if len(key) == 0 { return l }
	return nil
}
func (l *leafKV) entryAt(key string) itrie {
	if key == l.key_ { return l }
	return nil
}
func (l *leafV) foreach(prefix string, f func(string, Value)) {
	f(prefix, l.val_)
}
func (l *leafKV) foreach(prefix string, f func(string, Value)) {
	f(prefix + l.key_, l.val_)
}
func (l *leafV) withsubs(start uint, end uint, fn func(byte, itrie)) {}
func (l *leafKV) key() string { return l.key_ }
func (l *leafV) count() int { return 1 }
func (l *leafV) occupied() int { return 0 }
func (l *leafV) expanse() expanse_t { return expanse0() }
func (l *leafV) expanseWithout(byte) expanse_t { return expanse0() }


/*
 bag_t

 A bag is an ordered collection of sub-tries.  It may hold a single value itself.
 When a bag has 4 or more elements, it may be promoted to a span.  If it has 8 or more elements,
 it will be promoted to a span or a bitmap.
*/
type bag_ struct {
	entry_
	cb [maxBagSize]byte
	count_ int
	sub []itrie
}
type bagK struct {
	entryK
	bag_
}
type bagV struct {
	entryV
	bag_
}
type bagKV struct {
	entryKV
	bag_
}
func (b *bag_) printCBs(prefix string) {
	fmt.Printf("%s: bag.cb = [", prefix)
	for i := 0; i < len(b.sub); i++ {
		fmt.Printf(" %d(%c) ", b.cb[i], b.cb[i])
	}
	fmt.Printf("]\n")
}
func makeBag(key string, val Value, full bool) (*bag_, itrie) {
	if len(key) > 0 {
		if full {
			Cumulative[kBagKV]++
			b := new(bagKV)
			b.key_ = str(key); b.val_ = val; b.count_ = 1
			return &b.bag_, b
		}
		Cumulative[kBagK]++
		b := new(bagK)
		b.key_ = str(key)
		return &b.bag_, b
	}
	if full {
		Cumulative[kBagV]++
		b := new(bagV)
		b.val_ = val; b.count_ = 1
		return &b.bag_, b
	}
	Cumulative[kBag_]++
	b := new(bag_)
	return b, b
}
func (b *bag_) init1(cb byte, sub itrie) {
	b.cb[0] = cb
	b.sub = make([]itrie, 1)
	b.sub[0] = sub
	b.count_ += sub.count()
}	
func bag1(key string, val Value, full bool, cb byte, sub itrie) itrie {
	b, t := makeBag(key, val, full)
	b.init1(cb, sub)
	return t
}
func (b *bag_) init2(cb0, cb1 byte, sub0, sub1 itrie) {
	if cb1 < cb0 { cb0, cb1 = cb1, cb0; sub0, sub1 = sub1, sub0 }
	b.cb[0] = cb0; b.cb[1] = cb1 
	b.sub = make([]itrie, 2)
	b.sub[0] = sub0; b.sub[1] = sub1
	b.count_ += sub0.count() + sub1.count()
}
func bag2(key string, val Value, full bool, cb0, cb1 byte, sub0, sub1 itrie) itrie {
	b, t := makeBag(key, val, full)
	b.init2(cb0, cb1, sub0, sub1)
	return t
}
func (b *bag_) fillWith(t itrie, cb byte, l itrie) {
	b.sub = make([]itrie, t.occupied()+1)
	index := 0
	add := func(cb byte, t itrie) {	b.sub[index] = t; b.cb[index] = cb; index++ }
	t.withsubs(0, uint(cb), add)
	add(cb, l)
	t.withsubs(uint(cb)+1, 256, add)
	b.count_ = t.count() + 1
}
/*
 Constructs a new bag with the contents of t and l, where l is always a leaf.  It is known
 that l starts a new sub-trie -- t does not have a sub-trie at critical byte cb.
*/
func bag(t itrie, cb byte, l itrie) itrie {
	b, r := makeBag(t.key(), t.val(), t.hasVal())
	b.fillWith(t, cb, l)
	return r
}
/*
 Constructs a new bag with the contents of t, minus the sub-trie at critical bit cb.  It
 is expected that any sub-trie at cb is a leaf.
*/
func (b *bag_) fillWithout(t itrie, e expanse_t, without byte) {
	b.sub = make([]itrie, t.occupied()-1)
	index := 0
	add := func(cb byte, t itrie) { b.sub[index] = t; b.cb[index] = cb; index++ }
	t.withsubs(uint(e.low), uint(without), add)
	t.withsubs(uint(without+1), uint(e.high)+1, add)
	b.count_ = t.count() - 1
}
func bagWithout(t itrie, e expanse_t, without byte) itrie {
	if len(t.key()) > 0 {
		if t.hasVal() {
			b := new(bagKV)
			b.key_ = t.key(); b.val_ = t.val()
			b.fillWithout(t, e, without)
			return b
		}
		b := new(bagK)
		b.key_ = t.key()
		b.fillWithout(t, e, without)
		return b
	}
	if t.hasVal() {
		b := new(bagV)
		b.val_ = t.val()
		b.fillWithout(t, e, without)
		return b
	}
	b := new(bag_)
	b.fillWithout(t, e, without)
	return b
}
func (b *bag_) copy(t *bag_) {
	b.count_ = t.count_
	copy(b.cb[:len(t.sub)], t.cb[:len(t.sub)])
	b.sub = make([]itrie, len(t.sub))
	copy(b.sub, t.sub)
}
func (b *bag_) modify(incr, i int, sub itrie) itrie {
	n := new(bag_)
	n.copy(b); n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bagK) modify(incr, i int, sub itrie) itrie {
	n := new(bagK)
	n.key_ = b.key_;
	n.copy(&b.bag_); n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bagV) modify(incr, i int, sub itrie) itrie {
	n := new(bagV)
	n.val_ = b.val_
	n.copy(&b.bag_); n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bagKV) modify(incr, i int, sub itrie) itrie {
	n := new(bagKV)
	n.key_ = b.key_; n.val_ = b.val_;
	n.copy(&b.bag_); n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bag_) cloneWithKey(key string) itrie {
	n := new(bagK)
	n.bag_ = *b; n.key_ = str(key)
	return n
}
func (b *bagV) cloneWithKey(key string) itrie {
	n := new(bagKV)
	n.bag_ = b.bag_; n.key_ = str(key); n.val_ = b.val_
	return n
}
func (b *bagKV) cloneWithKey(key string) itrie {
	n := new(bagKV)
	n.bag_ = b.bag_; n.key_ = str(key); n.val_ = b.val_
	return n
}	
func (b *bag_) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := new(bagKV)
	n.bag_ = *b; n.key_ = str(key); n.val_ = val; n.count_++
	return n, 1
}
func (b *bagV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := new(bagKV)
	n.bag_ = b.bag_; n.key_ = str(key); n.val_ = val
	return n, 0
}
func (b *bagKV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := new(bagKV)
	n.bag_ = b.bag_; n.key_ = str(key); n.val_ = val
	return n, 0
}
func (b *bag_) withoutValue() (itrie, int) {
	return b, 0
}
func (b *bagV) withoutValue() (itrie, int) {
	n := b.bag_
	return &n, 1
}
func (b *bagKV) withoutValue() (itrie, int) {
	n := new(bagK)
	n.key_ = b.key_; n.bag_ = b.bag_
	return n, 1
}
func (n *bag_) with_(b *bag_, size, i int, cb byte, key string, val Value) int {
	n.count_ = b.count_
	n.sub = make([]itrie, size)
	if size > maxBagSize {
		panic(fmt.Sprintf("Don't make bag's with more than %d elts.", maxBagSize))
	}
	copy(n.cb[:i], b.cb[:i])
	copy(n.sub[:i], b.sub[:i])
	src, dst, added := i, i, 0
	if size == len(b.sub) {
		// We're modifying an existing sub-trie
		if cb != b.cb[i] { panic("We should be modifying a sub-trie") }
		n.cb[dst] = cb
		n.sub[dst], added = assoc(b.sub[src], key, val); dst++; src++
	} else {
		n.cb[dst] = cb
		n.sub[dst], added = assoc(nil, key, val); dst++
	}
	copy(n.cb[dst:], b.cb[src:])
	copy(n.sub[dst:], b.sub[src:])
	n.count_ = b.count_ + added
	return added
}
func (b *bag_) with(size, i int, cb byte, key string, val Value) (itrie, int) {
	n := new(bag_)
	added := n.with_(b, size, i, cb, key, val)
	return n, added
}
func (b *bagK) with(size, i int, cb byte, key string, val Value) (itrie, int) {
	n := new(bagK)
	n.key_ = b.key_
	added := n.with_(&b.bag_, size, i, cb, key, val)
	return n, added
}
func (b *bagV) with(size, i int, cb byte, key string, val Value) (itrie, int) {
	n := new(bagV)
	n.val_ = b.val_
	added := n.with_(&b.bag_, size, i, cb, key, val)
	return n, added
}
func (b *bagKV) with(size, i int, cb byte, key string, val Value) (itrie, int) {
	n := new(bagKV)
	n.key_ = b.key_; n.val_ = b.val_
	added := n.with_(&b.bag_, size, i, cb, key, val)
	return n, added
}
func (b *bag_) find(cb byte) (int, bool) {
	// Even though it's sorted, since len <= 7, it's almost certainly not worth it to
	// binary search.  We can still take advantage of early out.
	for i := 0; i < len(b.sub); i++ {
		if cb < b.cb[i] { return i, false }
		if cb == b.cb[i] { return i, true }
	}
	return len(b.sub), false
}
func (b *bag_) assoc(t itrie, prefix string, cb byte, rest string, val Value) (itrie, int) {
	i, found := b.find(cb)
	size := len(b.sub)
	if !found {
		size++
		// Determine whether we're a bag, span, or a bitmap
		if size >= minSpanSize {
			e := b.expanse().with(cb)
			if spanOK(e, size) {
				// Prefer a span, even if we're small enough to stay a bag
				return span(t, e, cb, leaf(rest, val)), 1
			}
		}
		if size > maxBagSize {
			// Prefer a bitmap
			return bitmap(t, cb, leaf(rest, val)), 1
		}
	}
	switch n := t.(type) {
	case *bag_: return n.with(size, i, cb, rest, val)
	case *bagK: return n.with(size, i, cb, rest, val)
	case *bagV: return n.with(size, i, cb, rest, val)
	case *bagKV: return n.with(size, i, cb, rest, val)
	}
	panic(fmt.Sprintf("unknown bag subtype: %T", t))
}
func (b *bag_) without(t itrie, key string) (itrie, int) {
	return b.without_(t, key, 0)
}
func (b *bagK) without(t itrie, key string) (itrie, int) {
	crit, _ := findcb(key, b.key_)
	if crit <= len(b.key_) {
		// we don't have the element being removed
		return t, 0
	}
	return b.bag_.without_(t, key, crit)
}
func (b *bagV) without(t itrie, key string) (itrie, int) {
	if len(key) == 0 {
		if len(b.sub) == 1 {
			// collapse this node to it's only child.
			key = string(b.cb[0]) + b.sub[0].key()
			return b.sub[0].cloneWithKey(key), 1
		}
		return t.withoutValue()
	}
	return b.bag_.without_(t, key, 0)
}
func (b *bagKV) without(t itrie, key string) (itrie, int) {
	crit, match := findcb(key, b.key_)
	if crit < len(b.key_) {
		// we don't have the element being removed
		return t, 0
	}
	if match {
		if len(b.sub) == 1 {
			// collapse this node to it's only child.
			key += string(b.cb[0]) + b.sub[0].key()
			return b.sub[0].cloneWithKey(key), 1
		}
		return t.withoutValue()
	}
	return b.bag_.without_(t, key, crit)
}
func (b *bag_) without_(t itrie, key string, crit int) (itrie, int) {
	_, cb, rest := splitKey(key, crit)
	i, found := b.find(cb)
	if !found {
		// we don't have the element being removed
		return b, 0
	}
	n, less := without(b.sub[i], rest)
	if n == nil {
		// We removed a leaf -- shrink our sub-tries & possibly turn into a leaf.
		last := len(b.sub)-1
		if last == 0 {
			if !t.hasVal() {
				panic("we should have a value if we have no sub-tries.")
			}
			return leaf(t.key(), t.val()), less
		} else if last == 1 && !t.hasVal() {
			o := 1 - i
			key = t.key() + string(b.cb[o]) + b.sub[o].key()
			return b.sub[o].cloneWithKey(key), less
		}
		e := b.expanse()
		if last >= minSpanSize {
			e = b.expanseWithout(cb)
			if spanOK(e, last) {
				// We can be a span
				return spanWithout(t, e, cb), less
			}
		}
		// Still a bag.
		return bagWithout(t, e, cb), less
	}
	if less == 0 {
		if n != b.sub[i] { panic("Shouldn't create a new node without changes.") }
		return t, 0
	}
	return t.modify(-1, i, n), less
}
func (b *bagKV) entryAt(key string) itrie {
	crit, match := findcb(key, b.key_)
	if match { return b }
	if crit >= len(key) { return nil }
	_, cb, rest := splitKey(key, crit)
	i, found := b.find(cb)
	if !found { return nil }
	return b.sub[i].entryAt(rest)
}
func (b *bagV) entryAt(key string) itrie {
	if len(key) == 0 { return b }
	_, cb, rest := splitKey(key, 0)
	i, found := b.find(cb)
	if !found { return nil }
	return b.sub[i].entryAt(rest)
}
func (b *bagK) entryAt(key string) itrie {
	crit, match := findcb(key, b.key_)
	if match { return nil }
	if crit >= len(key) { return nil }
	_, cb, rest := splitKey(key, crit)
	i, found := b.find(cb)
	if !found { return nil }
	return b.sub[i].entryAt(rest)
}
func (b *bag_) entryAt(key string) itrie {
	if len(key) == 0 { return nil }
	_, cb, rest := splitKey(key, 0)
	i, found := b.find(cb)
	if !found { return nil }
	return b.sub[i].entryAt(rest)
}
func (b *bagKV) foreach(prefix string, f func(key string, val Value)) {
	prefix += b.key_
	f(prefix, b.val_)
	b.bag_.foreach(prefix, f)
}
func (b *bagV) foreach(prefix string, f func(key string, val Value)) {
	f(prefix, b.val_)
	b.bag_.foreach(prefix, f)
}
func (b *bagK) foreach(prefix string, f func(key string, val Value)) {
	prefix += b.key_
	b.bag_.foreach(prefix, f)
}
func (b *bag_) foreach(prefix string, f func(key string, val Value)) {
	for i, sub := range b.sub {
		sub.foreach(prefix + string(b.cb[i]), f)
	}
}
func (b *bag_) withsubs(start, end uint, f func(byte, itrie)) {
	for i := 0; i < len(b.sub); i++ {
		if uint(b.cb[i]) < start { continue }
		if uint(b.cb[i]) >= end { break }
		f(b.cb[i], b.sub[i])
	}
}
func (b *bag_) count() int { return b.count_ }
func (b *bag_) occupied() int { return len(b.sub) }
func (b *bag_) expanse() expanse_t { return expanse(b.cb[0], b.cb[len(b.sub)-1]) }
func (b *bag_) expanseWithout(cb byte) expanse_t {
	if len(b.sub) == 0 { panic("Shouldn't have an empty bag.") }
	if len(b.sub) > 1 {
		last := len(b.sub)-1
		if cb == b.cb[0] { return expanse(b.cb[1], b.cb[last]) }
		if cb == b.cb[last] { return expanse(b.cb[0], b.cb[last-1]) }
	} else if cb == b.cb[0] { return expanse0() }
	return b.expanse()
}

/*
 span_t

 A span is a trie node that simply stores an array of sub-tries, where the critical byte
 for the sub-trie is index+start.  Range's are only used for sub-tries with high density.
*/
type span_ struct {
	entry_
	start byte
	occupied_ uint16
	count_ int
	sub []itrie
}
type spanK struct {
	entryK
	span_
}
type spanV struct {
	entryV
	span_
}
type spanKV struct {
	entryKV
	span_
}
func makeSpan(e expanse_t, key string, val Value, full bool) (s *span_, t itrie) {
	if len(key) > 0 {
		if full {
			Cumulative[kSpanKV]++
			n := new(spanKV)
			n.key_ = key; n.val_ = val; n.count_ = 1
			s, t = &n.span_, n
		} else {
			Cumulative[kSpanK]++
			n := new(spanK)
			n.key_ = key
			s, t = &n.span_, n
		}
	} else {
		if full {
			Cumulative[kSpanV]++
			n := new(spanV)
			n.val_ = val; n.count_ = 1
			s, t = &n.span_, n
		} else {
			Cumulative[kSpan_]++
			n := new(span_)
			s, t = n, n
		}
	}
	s.start = e.low
	s.sub = make([]itrie, int(e.size))
	return
}
/*
 Constructs a new span with th contents of t and l, where l is always a leaf.  It is known
 that l starts a new sub-trie -- t does not have a sub-trie at critical byte cb.
*/
func span(t itrie, e expanse_t, cb byte, l itrie) itrie {
	s, r := makeSpan(e, t.key(), t.val(), t.hasVal())
	add := func(cb byte, t itrie) {
		s.sub[cb - s.start] = t
	}
	t.withsubs(0, uint(cb), add)
	add(cb, l)
	t.withsubs(uint(cb+1), 256, add)
	s.count_ = t.count() + 1
	s.occupied_ = uint16(t.occupied() + 1)
	return r
}
/*
 Constructs a new span with the contents of t, minus the sub-trie at critical byte cb.  It
 is expected that any sub-trie at cb is a leaf.
*/
func spanWithout(t itrie, e expanse_t, without byte) itrie {
	s, r := makeSpan(e, t.key(), t.val(), t.hasVal())
	add := func(cb byte, t itrie) { s.sub[cb - s.start] = t	}
	t.withsubs(uint(e.low), uint(without), add)
	t.withsubs(uint(without+1), uint(e.high)+1, add)
	s.count_ = t.count() - 1
	s.occupied_ = uint16(t.occupied() - 1)
	return r
}

func (s *span_) copy(t *span_) {
	s.start = t.start
	s.count_ = t.count_
	s.sub = make([]itrie, len(t.sub))
	copy(s.sub, t.sub)
}
func (s *span_) cloneWithKey(key string) itrie {
	n := new(spanK)
	n.span_ = *s; n.key_ = str(key)
	return n
}
func (s *spanV) cloneWithKey(key string) itrie {
	n := new(spanKV)
	n.span_ = s.span_; n.key_ = str(key); n.val_ = s.val_
	return n
}
func (s *spanKV) cloneWithKey(key string) itrie {
	n := new(spanKV)
	n.span_ = s.span_; n.key_ = str(key); n.val_ = s.val_
	return n
}
func (s *span_) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := new(spanKV)
	n.span_ = *s; n.key_ = str(key); n.val_ = val
	return n, 1
}
func (s *spanV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := new(spanKV)
	n.span_ = s.span_; n.key_ = str(key); n.val_ = val
	return n, 0
}
func (s *spanKV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := new(spanKV)
	n.span_ = s.span_; n.key_ = str(key); n.val_ = val
	return n, 0
}
func (s *span_) modify(incr, i int, sub itrie) itrie {
	n := new(span_)
	n.copy(s); n.count_ += incr; n.sub[i] = sub
	return n
}
func (s *spanK) modify(incr, i int, sub itrie) itrie {
	n := new(spanK)
	n.key_ = s.key_
	n.copy(&s.span_); n.count_ += incr; n.sub[i] = sub
	return n
}
func (s *spanV) modify(incr, i int, sub itrie) itrie {
	n := new(spanV)
	n.val_ = s.val_
	n.copy(&s.span_); n.count_ += incr; n.sub[i] = sub
	return n
}
func (s *spanKV) modify(incr, i int, sub itrie) itrie {
	n := new(spanKV)
	n.key_ = s.key_; n.val_ = s.val_
	n.copy(&s.span_); n.count_ += incr; n.sub[i] = sub
	return n
}
func (s *span_) withoutValue() (itrie, int) {
	return s, 0
}
func (s *spanV) withoutValue() (itrie, int) {
	n := new(span_)
	*n = s.span_; n.count_--
	return n, 1
}
func (s *spanKV) withoutValue() (itrie, int) {
	n := new(spanK)
	n.span_ = s.span_; n.key_ = s.key_; n.count_--
	return n, 1
}
func (n *span_) with_(s *span_, e expanse_t, cb byte, key string, val Value) int {
	if e.low > s.start { panic("new start must be <= old start") }
	if int(e.size) < len(s.sub) { panic("new size must be >= old size") }
	n.start = e.low; n.count_ = s.count_; n.occupied_ = s.occupied_
	n.sub = make([]itrie, int(e.size))
	copy(n.sub[s.start - n.start:], s.sub)
	i, added := int(cb - n.start), 0
	o := n.sub[i]
	n.sub[i], added = assoc(o, key, val)
	n.count_ += added
	if o == nil { n.occupied_++ }
	return added
}
func (s *span_) with(e expanse_t, cb byte, key string, val Value) (itrie, int) {
	n := new(span_)
	added := n.with_(s, e, cb, key, val)
	return n, added
}
func (s *spanK) with(e expanse_t, cb byte, key string, val Value) (itrie, int) {
	n := new(spanK)
	n.key_ = s.key_
	added := n.with_(&s.span_, e, cb, key, val)
	return n, added
}
func (s *spanV) with(e expanse_t, cb byte, key string, val Value) (itrie, int) {
	n := new(spanV)
	n.val_ = s.val_
	added := n.with_(&s.span_, e, cb, key, val)
	return n, added
}
func (s *spanKV) with(e expanse_t, cb byte, key string, val Value) (itrie, int) {
	n := new(spanKV)
	n.key_ = s.key_; n.val_ = s.val_
	added := n.with_(&s.span_, e, cb, key, val)
	return n, added
}
func (s *span_) assoc(t itrie, prefix string, cb byte, rest string, val Value) (itrie, int) {
	// Update expanse
	e0 := s.expanse()
	e := e0.with(cb)
	
	if e.size > e0.size {
		// Figure out if we're a span, a bag, or a bitmap.
		count := int(s.occupied_)+1
		if !spanOK(e, count) {
			// We're not a span.
			if count <= maxBagSize {
				return bag(t, cb, leaf(rest, val)), 1
			}
			// Prefer a bitmap
			return bitmap(t, cb, leaf(rest, val)), 1
		}
	}
	
	// Prefer a span -- the code below handles the case of adding a new child, or
	// overwriting an existing one.
	switch n := t.(type) {
	case *span_: return n.with(e, cb, rest, val)
	case *spanK: return n.with(e, cb, rest, val)
	case *spanV: return n.with(e, cb, rest, val)
	case *spanKV: return n.with(e, cb, rest, val)
	}
	panic(fmt.Sprintf("unknown span subtype: %T", t))
}
func (s *span_) firstAfter(i int) byte {
	i++
	for ; i < len(s.sub); i++ {
		if s.sub[i] != nil { return byte(i) }
	}
	panic("no further occupied elements in span")
}
func (s *span_) lastBefore(i int) byte {
	i--
	for ; i >= 0; i-- {
		if s.sub[i] != nil { return byte(i) }
	}
	panic("no prior occupied elements in span")
}
func (s *span_) expanseWithout(cb byte) expanse_t {
	e := s.expanse()
	if cb == e.low {
		d := s.firstAfter(0)
		e.low += byte(d)
		e.size -= uint16(d)
	}
	if cb == e.high {
		d := s.lastBefore(len(s.sub)-1)
		e.high = e.low + byte(d)
		e.size = uint16(d+1)
	}
	return e
}
func (s *span_) without(t itrie, key string) (itrie, int) {
	return s.without_(t, key, 0)
}
func (s *spanK) without(t itrie, key string) (itrie, int) {
	crit, match := findcb(key, s.key_)
	if crit < len(s.key_) {
		// we don't have the element being removed
		return t, 0
	}

	if match {
		// we don't have the element being removed
		return t, 0
	}
	return s.without_(t, key, crit)
}
func (s *spanV) without(t itrie, key string) (itrie, int) {
	if len(key) == 0 {
		if s.occupied_ == 1 {
			for i, c := range s.sub {
				// collapse this node to it's only child.
				if c == nil { continue }
				key = string(s.start+byte(i)) + c.key()
				return c.cloneWithKey(key), 1
			}
			panic("should have found a non-nil sub-trie.")
		}
		return t.withoutValue()
	}
	return s.without_(t, key, 0)
}
func (s *spanKV) without(t itrie, key string) (itrie, int) {
	crit, match := findcb(key, s.key_)
	if crit < len(s.key_) {
		// we don't have the element being removed
		return t, 0
	}

	if match {
		if s.occupied_ == 1 {
			for i, c := range s.sub {
				// collapse this node to it's only child.
				if c == nil { continue }
				key += string(s.start+byte(i)) + c.key()
				return c.cloneWithKey(key), 1
			}
			panic("should have found a non-nil sub-trie.")
		}
		return t.withoutValue()
	}
	return s.without_(t, key, crit)
}
func (s *span_) without_(t itrie, key string, crit int) (itrie, int) {
	_, cb, rest := splitKey(key, crit)
	if cb < s.start {
		// we don't have the element being removed
		return t, 0
	}
	i := cb - s.start
	if int(i) >= len(s.sub) {
		// we don't have the element being removed
		return t, 0
	}
	n, less := without(s.sub[i], rest)
	if n == nil {
		// We removed a leaf -- shrink our children & possibly turn into a bag or leaf.
		occupied := s.occupied_ - 1
		// We shouldn't actually let spans get small enough to hit either of the next
		// two cases
		if occupied == 0 {
			if !t.hasVal() { panic("we should have a value if we have no sub-tries.") }
			return leaf(t.key(), t.val()), less
		} 
		if occupied == 1 && !t.hasVal() {
			o := 0
			for ; o < len(s.sub); o++ {
				if byte(o) != i && s.sub[o] != nil { break }
			}
			if o >= len(s.sub) { panic("We should have another valid sub-trie") }
			key = t.key() + string(cb) + s.sub[o].key()
			return s.sub[o].cloneWithKey(key), less
		}
		e := s.expanse()
		if occupied >= minSpanSize {
			e = s.expanseWithout(cb)
			if spanOK(e, int(occupied)) {
				// We can stay a span
				return spanWithout(t, e, cb), less
			}
		}
		if occupied <= maxBagSize {
			// We should become a bag
			return bagWithout(t, e, cb), less
		}
		// Looks like we're a bitmap
		return bitmapWithout(t, e, cb), less
	}
	if less == 0 {
		if n != s.sub[i] { panic("Shouldn't make a new node without changes.") }
		return t, 0
	}
	return t.modify(-1, int(i), n), less
}
func (s *span_) entryAt(key string) itrie {
	if len(key) == 0 { return nil }
	_, cb, rest := splitKey(key, 0)
	i := int(cb) - int(s.start)
	if i >= 0 && i < len(s.sub) {
		if s.sub[i] != nil { return s.sub[i].entryAt(rest) }
	}
	return nil
}
func (s *spanK) entryAt(key string) itrie {
	crit, _ := findcb(key, s.key_)
	if crit >= len(key) { return nil }
	_, cb, rest := splitKey(key, crit)
	i := int(cb) - int(s.start)
	if i >= 0 && i < len(s.sub) {
		if s.sub[i] != nil { return s.sub[i].entryAt(rest) }
	}
	return nil
}
func (s *spanV) entryAt(key string) itrie {
	if len(key) == 0 { return s }
	_, cb, rest := splitKey(key, 0)
	i := int(cb) - int(s.start)
	if i >= 0 && i < len(s.sub) {
		if s.sub[i] != nil { return s.sub[i].entryAt(rest) }
	}
	return nil
}
func (s *spanKV) entryAt(key string) itrie {
	crit, match := findcb(key, s.key_)
	if match { return s }
	if crit >= len(key) { return nil }
	_, cb, rest := splitKey(key, crit)
	i := int(cb) - int(s.start)
	if i >= 0 && i < len(s.sub) {
		if s.sub[i] != nil { return s.sub[i].entryAt(rest) }
	}
	return nil
}
func (s *span_) foreach(prefix string, f func(string, Value)) {
	for i, t := range s.sub {
		if t != nil {
			t.foreach(prefix + string(s.start+byte(i)), f)
		}
	}
}
func (s *spanK) foreach(prefix string, f func(string, Value)) {
	prefix += s.key_
	s.span_.foreach(prefix, f)
}
func (s *spanV) foreach(prefix string, f func(string, Value)) {
	f(prefix, s.val_)
	s.span_.foreach(prefix, f)
}
func (s *spanKV) foreach(prefix string, f func(string, Value)) {
	prefix += s.key_
	f(prefix, s.val_)
	s.span_.foreach(prefix, f)
}

func (s *span_) withsubs(start, end uint, f func(byte, itrie)) {
	start = uint(min(max(0, int(start) - int(s.start)), len(s.sub)))
	end = uint(min(max(0, int(end) - int(s.start)), len(s.sub)))
	if start >= end { return }
	for i, t := range s.sub[start:end] {
		if t == nil { continue }
		cb := s.start + byte(start) + byte(i)
		f(cb, t)
	}
}
func (s *span_) count() int { return s.count_ }
func (s *span_) occupied() int { return int(s.occupied_) }
func (s *span_) expanse() expanse_t { return expanse(s.start, s.start+byte(len(s.sub)-1)) }

/*
 bitmap_t

 A bitmap is a trie node that uses a bitmap to track which of it's children are occupied.
*/
type bitmap_t struct {
	key_ string
	val_ Value
	count_ int
	full bool
	off [4]uint8
	bm [4]uint64
	sub []itrie
}
/*
 population count implementation taken from http://www.wikipedia.org/wiki/Hamming_weight
*/
const m1  = 0x5555555555555555
const m2  = 0x3333333333333333
const m4  = 0x0f0f0f0f0f0f0f0f
const m8  = 0x00ff00ff00ff00ff
const m16 = 0x0000ffff0000ffff
const m32 = 0x00000000ffffffff
const h01 = 0x0101010101010101
func countbits(bits uint64) byte {
	bits -= (bits >> 1) & m1
	bits = (bits & m2) + ((bits >> 2) & m2)
	bits = (bits + (bits >> 4)) & m4
	return byte((bits*h01)>>56)
}
func reverse(bits uint64) uint64 {
	bits = ((bits >>  1) &  m1) | ((bits &  m1) <<  1)
	bits = ((bits >>  2) &  m2) | ((bits &  m2) <<  2)
	bits = ((bits >>  4) &  m4) | ((bits &  m4) <<  4)
	bits = ((bits >>  8) &  m8) | ((bits &  m8) <<  8)
	bits = ((bits >> 16) & m16) | ((bits & m16) << 16)
	bits = ((bits >> 32) & m32) | ((bits & m32) << 32)
	return bits
}
func bitpos(ch uint) (int, uint64) {
	return int((ch >> 6)), uint64(1) << (ch & 0x3f)
}
func (b *bitmap_t) setbit(w int, bit uint64) {
	b.bm[w] |= bit
	for ; w < 3; w++ { b.off[w+1] += 1 }
}
func (b *bitmap_t) clearbit(w int, bit uint64) {
	b.bm[w] &= ^bit
	for ; w < 3; w++ { b.off[w+1] -= 1 }
}
func (b *bitmap_t) isset(w int, bit uint64) bool {
	return b.bm[w] & bit != 0
}
func (b *bitmap_t) indexOf(w int, bit uint64) int {
	return int(countbits(b.bm[w] & (bit-1)) + b.off[w])
}
func minbit(bm uint64) byte {
	bit := bm ^ (bm & (bm-1))
	return countbits(bit-1)
}
func maxbit(bm uint64) byte {
	bm = reverse(bm)
	bit := bm ^ (bm & (bm-1))
	return byte(63) - countbits(bit-1)
}
func (b *bitmap_t) min() byte {
	for w, bm := range b.bm {
		if bm == 0 { continue }
		return minbit(bm) + byte(64*w)
	}
	panic("Didn't find any bits set in bitmap")
}
func (b *bitmap_t) max() byte {
	for w := 3; w >= 0; w-- {
		if b.bm[w] == 0 { continue }
		return maxbit(b.bm[w]) + byte(64*w)
	}
	panic("Didn't find any bits set in bitmap")
}
func (b *bitmap_t) lastBefore(cb byte) byte {
	w, bit := bitpos(uint(cb))
	mask := bit - 1
	bm := b.bm[w] & mask
	if bm != 0 { return maxbit(bm) }
	for ; w >= 0; w-- {
		if b.bm[w] == 0 { continue }
		return maxbit(b.bm[w])
	}		
	return cb
}
func (b *bitmap_t) firstAfter(cb byte) byte {
	w, bit := bitpos(uint(cb))
	mask := ^((bit - 1) | bit)
	bm := b.bm[w] & mask
	if bm != 0 { return minbit(bm) }
	for ; w < 4; w++ {
		if b.bm[w] == 0 { continue }
		return minbit(b.bm[w])
	}
	return cb
}
func makeBitmap(size int, key string, val Value, full bool) *bitmap_t {
	if len(key) > 0 {
		if full { Cumulative[kBitmapKV]++ } else { Cumulative[kBitmapK]++ }
	} else {
		if full { Cumulative[kBitmapV]++ } else { Cumulative[kBitmap_]++ }
	}
	bm := new(bitmap_t); bm.key_ = key; bm.val_ = val; bm.full = full
	bm.sub = make([]itrie, size)
	return bm
}
/*
 Constructs a new bitmap with the contents of t and l, where l is always a leaf.  It is known
 that l starts a new sub-trie -- t does not have a sub-trie at critical byte cb.
*/
func bitmap(t itrie, cb byte, l itrie) *bitmap_t {
	bm := makeBitmap(t.occupied()+1, t.key(), t.val(), t.hasVal())
	index := 0
	add := func(cb byte, t itrie) {
		w, bit := bitpos(uint(cb))
		bm.sub[index] = t; bm.setbit(w, bit); index++
	}
	t.withsubs(0, uint(cb), add)
	add(cb, l)
	t.withsubs(uint(cb+1), 256, add)
	bm.count_ = t.count() + 1
	return bm
}
/*
 Constructs a new bitmap with the contents of t, minus the sub-trie at critical byte cb.  It
 is expected that any sub-trie at cb is a leaf.
*/
func bitmapWithout(t itrie, e expanse_t, without byte) *bitmap_t {
	bm := makeBitmap(t.occupied()-1, t.key(), t.val(), t.hasVal())
	index := 0
	add := func(cb byte, t itrie) { 
		bm.sub[index] = t; bm.setbit(bitpos(uint(cb))); index++
	}
	t.withsubs(uint(e.low), uint(without), add)
	t.withsubs(uint(without+1), uint(e.high)+1, add)
	bm.count_ = t.count() - 1
	return bm
}
func (b *bitmap_t) clone() *bitmap_t {
	n := new(bitmap_t)
	*n = *b
	n.sub = make([]itrie, len(b.sub))
	copy(n.sub, b.sub)
	return n
}
func (b *bitmap_t) cloneWithKey(key string) itrie {
	n := b.clone()
	n.key_ = str(key)
	return n
}
func (b *bitmap_t) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := b.clone()
	n.key_ = str(key); n.val_ = val; n.full = true
	if !b.full { n.count_++; return n, 1 }
	return n, 0
}
func (b *bitmap_t) modify(incr, i int, sub itrie) itrie {
	n := b.clone()
	n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bitmap_t) withoutValue() (itrie, int) {
	if b.full {
		n := b.clone()
		n.val_ = nil; n.full = false; n.count_--
		return n, 1
	}
	return b, 0
}
func (b *bitmap_t) with(cb byte, key string, val Value) (itrie, int) {
	n := new(bitmap_t)
	*n = *b
	w, bit := bitpos(uint(cb))
	size := len(b.sub)
	if !b.isset(w, bit) { size++ }
	n.sub = make([]itrie, size)
	i := b.indexOf(w, bit)
	copy(n.sub[:i], b.sub[:i])
	src, dst, added := i, i, 0
	if b.isset(w, bit) {
		// replace existing sub-trie
		n.sub[dst], added = assoc(b.sub[src], key, val); dst++; src++
	} else {
		n.sub[dst], added = assoc(nil, key, val); n.setbit(w, bit); dst++
	}
	copy(n.sub[dst:], b.sub[src:])
	n.count_ = b.count_ + added
	return n, added
}
func (b *bitmap_t) assoc(t itrie, prefix string, cb byte, rest string, val Value) (itrie, int) {
	// Figure out if we stay a bitmap or if we can become a span
	// we know we're too big to be a bag
	w, bit := bitpos(uint(cb))
	replace := b.isset(w, bit)
	if !replace {
		e := b.expanse().with(cb)
		count := len(b.sub)
		if spanOK(e, count+1) {
			// We can be a span
			return span(t, e, cb, leaf(rest, val)), 1
		}
	}
	// still a bitmap
	return b.with(cb, rest, val)
}
func (b *bitmap_t) without(t itrie, key string) (itrie, int) {
	crit, match := findcb(key, b.key_)
	if crit < len(b.key_) {
		// we don't have the element being removed
		return t, 0
	}

	if match {
		// we won't even check for the case of only 1 child in a bitmap -- it just
		// shouldn't happen
		return t.withoutValue()
	}

	_, cb, rest := splitKey(key, crit)
	w, bit := bitpos(uint(cb))
	if !b.isset(w, bit) {
		// we don't have the element being removed
		return t, 0
	}
	i := b.indexOf(w, bit)
	n, less := without(b.sub[i], rest)
	if n == nil {
		// We removed a leaf -- shrink our children & possibly turn into a bag or span.
		occupied := b.occupied() - 1
		e := b.expanseWithout(cb)
		if spanOK(e, occupied) {
			// We can be a span
			return spanWithout(t, e, cb), less
		}
		if occupied <= maxBagSize {
			// We should become a bag
			return bagWithout(t, e, cb), less
		}
		// We should stay a bitmap
		return bitmapWithout(t, e, cb), less
	}
	if less == 0 {
		if n != b.sub[i] { panic("Shouldn't create a new node without changes.") }
		return t, 0
	}
	return b.modify(-1, i, n), less
}
func (b *bitmap_t) entryAt(key string) itrie {
	crit, match := findcb(key, b.key_)
	if match && b.full { return b }
	if crit >= len(key) { return nil }
	_, cb, rest := splitKey(key, crit)
	w, bit := bitpos(uint(cb))
	if b.isset(w, bit) {
		index := b.indexOf(w, bit)
		return b.sub[index].entryAt(rest)
	}
	return nil
}
func (b *bitmap_t) foreach(prefix string, f func(string, Value)) {
	prefix += b.key_
	if b.full {
		f(prefix, b.val_)
	}
	b.withsubs(0, 256, func(cb byte, t itrie) {
		t.foreach(prefix + string(cb), f)
	})
}
func (b *bitmap_t) withsubs(start, end uint, f func(byte, itrie)) {
	sw, sbit := bitpos(start); sw = min(sw, len(b.bm))
	index := b.indexOf(sw, sbit)
	ew, ebit := bitpos(end); ew = min(ew, len(b.bm))
	mw := ew + 1
	if mw >= len(b.bm) { mw = len(b.bm)-1 }

	for i, bm := range b.bm[sw:mw] {
		w := i + sw
		for ; bm != 0; bm &= (bm-1) {
			bit := bm ^ (bm & (bm - 1))
			if w == sw && bit < sbit { 
				continue 
			}
			if w == ew && bit >= ebit {
				break
			}
			cb := countbits(bit-1) + byte(64*w)
			f(cb, b.sub[index])
			index++
		}
	}
}
func (b *bitmap_t) key() string { return b.key_ }
func (b *bitmap_t) hasVal() bool { return b.full }
func (b *bitmap_t) val() Value { return b.val_ }
func (b *bitmap_t) count() int { return b.count_ }
func (b *bitmap_t) occupied() int { return len(b.sub) }
func (b *bitmap_t) expanse() expanse_t { return expanse(b.min(), b.max()) }
func (b *bitmap_t) expanseWithout(cb byte) expanse_t {
	e := b.expanse()
	if cb == e.low {
		e.low = b.firstAfter(cb)
	}
	if cb == e.high {
		e.high = b.lastBefore(cb)
	}
	return expanse(e.low, e.high)
	
}

