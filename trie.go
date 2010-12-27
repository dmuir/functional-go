package immutable

import (
	"fmt"
)

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
func findcrit(a string, b string) (int, bool) {
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


func assoc(m itrie, key string, val Value) (itrie, int) {
	if m != nil {
		return m.assoc(key, val)
	}
	return leaf(key, val), 1
}

/*
 itrie.

 This is the interface of the internal trie nodes.
*/
type itrie interface {
	key() string
	hasVal() bool
	val() Value
	cloneWithKey(string) itrie
	cloneWithKeyValue(string, Value) (itrie, int)
	assoc(string, Value) (itrie, int)
	without(string) (itrie, int)
	entryAt(string) itrie
	count() int
	occupied() int
	expanse() expanse_t
	expanseWithout(byte) expanse_t
	inorder(string, chan Item)
	withsubs(start uint, end uint, fn func (byte, itrie))
	debugPrint(string)
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
	if d.t != nil {
		t, _ := d.t.without(key)
		return dict{t}
	}
	return nil
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
func (d dict) Iter() chan Item {
	ch := make(chan Item)
	if d.t != nil {
		go d.t.inorder("", ch)
	} else {
		go close(ch)
	}
	return ch
}

func Dict() IDict {
	return dict{nil}
}

/*
 leaf_t

 Internal node that contains only a key and a value.
*/
type leaf_t struct {
	key_ string
	val_ Value
}
func leaf(key string, val Value) *leaf_t {
	return &leaf_t{str(key), val}
}
func (l *leaf_t) clone() *leaf_t {
	return leaf(l.key_, l.val_)
}
func (l *leaf_t) cloneWithKey(key string) itrie {
	return leaf(key, l.val_)
}
func (l *leaf_t) cloneWithKeyValue(key string, val Value) (itrie, int) {
	return leaf(key, val), 0
}
func (l *leaf_t) assoc(key string, val Value) (itrie, int) {
	crit, match := findcrit(key, l.key_)
	if match {
		return l.cloneWithKeyValue(key, val)
	}

	prefix, cb, rest := splitKey(key, crit)
	_, _cb, _rest := splitKey(l.key_, crit)
	if crit == len(key) {
		return bag1(prefix, val, true, _cb, leaf(_rest, l.val_)), 1
	} else if crit == len(l.key_) {
		return bag1(prefix, l.val_, true, cb, leaf(rest, val)), 1
	}
	return bag2(prefix, nil, false, cb, _cb, leaf(rest, val), leaf(_rest, l.val_)), 1
}
func (l *leaf_t) without(key string) (itrie, int) {
	if key == l.key_ { return nil, 1 }
	return l, 0
}
func (l *leaf_t) entryAt(key string) itrie {
	if key == l.key_ { return l }
	return nil
}
func (l *leaf_t) inorder(prefix string, ch chan Item) {
	ch <- Item{prefix + l.key_, l.val_}
	if len(prefix) == 0 { close(ch) }
}
func (l *leaf_t) withsubs(start uint, end uint, fn func(byte, itrie)) {}
func (l *leaf_t) key() string { return l.key_ }
func (l *leaf_t) hasVal() bool { return true }
func (l *leaf_t) val() Value { return l.val_ }
func (l *leaf_t) count() int { return 1 }
func (l *leaf_t) occupied() int { return 0 }
func (l *leaf_t) expanse() expanse_t { return expanse0() }
func (l *leaf_t) expanseWithout(byte) expanse_t { return expanse0() }
func (l *leaf_t) debugPrint(prefix string) { fmt.Printf("%sleaf -- key: %s\n", prefix, l.key_) }


/*
 bag_t

 A bag is an ordered collection of sub-tries.  It may hold a single value itself.
 When a bag has 4 or more elements, it may be promoted to a span.  If it has 8 or more elements,
 it will be promoted to a span or a bitmap.
*/
type bag_t struct {
	key_ string
	val_ Value
	count_ int
	full bool
	crit [maxBagSize]byte
	sub []itrie
}
func bag1(key string, val Value, full bool, cb byte, sub itrie) *bag_t {
	b := new(bag_t)
	b.key_ = str(key); b.val_ = val; b.full = full
	if full { b.count_++ }
	b.crit[0] = cb
	b.sub = make([]itrie, 1)
	b.sub[0] = sub
	b.count_ += sub.count()
	return b
}
func bag2(key string, val Value, full bool, cb0, cb1 byte, sub0, sub1 itrie) *bag_t {
	b := new(bag_t)
	b.key_ = str(key); b.val_ = val; b.full = full
	if full { b.count_++ }
	if cb1 < cb0 { cb0, cb1 = cb1, cb0; sub0, sub1 = sub1, sub0 }
	b.crit[0] = cb0; b.crit[1] = cb1
	b.sub = make([]itrie, 2)
	b.sub[0] = sub0; b.sub[1] = sub1
	b.count_ += sub0.count() + sub1.count()
	return b
}
func makeBag(size int, key string, val Value, full bool) *bag_t {
	b := new(bag_t)
	b.key_ = key; b.val_ = val; b.full = full
	b.sub = make([]itrie, size)
	return b
}
/*
 Constructs a new bag with the contents of t and l, where l is always a leaf.  It is known
 that l starts a new sub-trie -- t does not have a sub-trie at critical byte cb.
*/
func bag(t itrie, cb byte, l *leaf_t) *bag_t {
	b := makeBag(t.occupied()+1, t.key(), t.val(), t.hasVal())
	index := 0
	add := func(cb byte, t itrie) {	
		b.sub[index] = t; b.crit[index] = cb; index++
	}
	t.withsubs(0, uint(cb), add)
	add(cb, l)
	t.withsubs(uint(cb)+1, 256, add)
	b.count_ = t.count() + 1
	return b
}
/*
 Constructs a new bag with the contents of t, minus the sub-trie at critical bit cb.  It
 is expected that any sub-trie at cb is a leaf.
*/
func bagWithout(t itrie, e expanse_t, without byte) *bag_t {
	b := makeBag(t.occupied() - 1, t.key(), t.val(), t.hasVal())
	index := 0
	add := func(cb byte, t itrie) { b.sub[index] = t; b.crit[index] = cb; index++ }
	t.withsubs(uint(e.low), uint(without), add)
	t.withsubs(uint(without+1), uint(e.high)+1, add)
	b.count_ = t.count() - 1
	return b
}
func (b *bag_t) clone() *bag_t {
	n := new(bag_t)
	*n = *b
	n.sub = make([]itrie, len(b.sub))
	copy(n.sub, b.sub)
	return n
}
func (b *bag_t) cloneWithKey(key string) itrie {
	n := b.clone()
	n.key_ = str(key)
	return n
}
func (b *bag_t) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := b.clone()
	n.key_ = str(key); n.val_ = val; n.full = true
	if !b.full { n.count_++; return n, 1 }
	return n, 0
}
func (b *bag_t) with(size, i int, cb byte, key string, val Value) (itrie, int) {
	n := new(bag_t)
	*n = *b
	n.sub = make([]itrie, size)
	if size > maxBagSize {
		panic(fmt.Sprintf("Don't make bag's with more than %d elts.", maxBagSize))
	}
	copy(n.crit[:i], b.crit[:i])
	copy(n.sub[:i], b.sub[:i])
	src, dst, added := i, i, 0
	if size == len(b.sub) {
		// We're modifying an existing sub-trie
		if cb != b.crit[i] { panic("We should be modifying a sub-trie") }
		n.sub[dst], added = b.sub[src].assoc(key, val); dst++; src++
	} else {
		n.crit[dst] = cb
		n.sub[dst], added = assoc(nil, key, val); dst++
	}
	copy(n.crit[dst:], b.crit[src:])
	copy(n.sub[dst:], b.sub[src:])
	n.count_ = b.count_ + added
	return n, added
}
func (b *bag_t) find(cb byte) (int, bool) {
	// Even though it's sorted, since len <= 7, it's almost certainly not worth it to
	// binary search.  We can still take advantage of early out.
	for i := 0; i < len(b.sub); i++ {
		if cb < b.crit[i] { return i, false }
		if cb == b.crit[i] { return i, true }
	}
	return len(b.sub), false
}

func (b *bag_t) assoc(key string, val Value) (itrie, int) {
	crit, match := findcrit(key, b.key_)
	if match {
		return b.cloneWithKeyValue(key, val)
	}
	
	prefix, cb, rest := splitKey(key, crit)
	_, _cb, _rest := splitKey(b.key_, crit)	
	if crit < len(b.key_) {
		return bag2(prefix, nil, false, cb, _cb,
			leaf(rest, val), b.cloneWithKey(_rest)), 1
	}
	i, found := b.find(cb)
	size := len(b.sub)
	if !found {
		size++
		// Determine whether we're a bag, span, or a bitmap
		if size >= minSpanSize {
			e := b.expanse().with(cb)
			if spanOK(e, size) {
				// Prefer a span, even if we're small enough to stay bag
				return span(b, e, cb, leaf(rest, val)), 1
			}
		}
		if size > maxBagSize {
			// Prefer a bitmap
			return bitmap(b, cb, leaf(rest, val)), 1
		}
	}
		
	// We still fit in a bag
	return b.with(size, i, cb, rest, val)
}

func (b *bag_t) without(key string) (itrie, int) {
	crit, match := findcrit(key, b.key_)
	if crit < len(b.key_) {
		// we don't have the element being removed
		return b, 0
	}

	if match {
		if len(b.sub) == 1 {
			// collapse this node to it's only child.
			key += string(b.crit[0]) + b.sub[0].key()
			return b.sub[0].cloneWithKey(key), 1
		}
		if !b.full {
			// Don't have the element
			return b, 0
		}
		n := b.clone();	n.val_ = nil; n.full = false; n.count_--
		return n, 1
	}

	_, cb, rest := splitKey(key, crit)
	i, found := b.find(cb)
	if !found {
		// we don't have the element being removed
		return b, 0
	}
	n, less := b.sub[i].without(rest)
	if n == nil {
		// We removed a leaf -- shrink our children & possibly turn into a leaf.
		last := len(b.sub)-1
		if last == 0 {
			if !b.full {
				panic("we should have a value if we have no sub-tries.")
			}
			return leaf(b.key_, b.val_), less
		} else if last == 1 && !b.full {
			o := 1 - i
			key = b.key_ + string(b.crit[o]) + b.sub[o].key()
			return b.sub[o].cloneWithKey(key), less
		}
		e := b.expanse()
		if last >= minSpanSize {
			e = b.expanseWithout(cb)
			if spanOK(e, last) {
				// We can be a span
				return spanWithout(b, e, cb), less
			}
		}
		// Still a bag.
		return bagWithout(b, e, cb), less
	}
	if less == 0 {
		if n != b.sub[i] { panic("Shouldn't create a new node without changes.") }
		return b, 0
	}
	b = b.clone(); b.sub[i] = n; b.count_ -= less
	return b, less
}
func (b *bag_t) entryAt(key string) itrie {
	crit, match := findcrit(key, b.key_)
	if match && b.full { return b }
	if crit >= len(key) { return nil }
	_, cb, rest := splitKey(key, crit)
	i, found := b.find(cb)
	if !found { return nil }
	return b.sub[i].entryAt(rest)
}
func (b *bag_t) inorder(prefix string, ch chan Item) {
	root := len(prefix) == 0
	prefix += b.key_
	if b.full {
		ch <- Item{prefix, b.val_}
	}
	for i := 0; i < len(b.sub); i++ {
		b.sub[i].inorder(prefix + string(b.crit[i]), ch)
	}
	if root { close(ch) }
}
func (b *bag_t) withsubs(start, end uint, f func(byte, itrie)) {
	for i := 0; i < len(b.sub); i++ {
		if uint(b.crit[i]) < start { continue }
		if uint(b.crit[i]) >= end { break }
		f(b.crit[i], b.sub[i])
	}
}
func (b *bag_t) key() string { return b.key_ }
func (b *bag_t) hasVal() bool { return b.full }
func (b *bag_t) val() Value { return b.val_ }
func (b *bag_t) count() int { return b.count_ }
func (b *bag_t) occupied() int { return len(b.sub) }
func (b *bag_t) expanse() expanse_t { return expanse(b.crit[0], b.crit[len(b.sub)-1]) }
func (b *bag_t) expanseWithout(cb byte) expanse_t {
	if len(b.sub) == 0 { panic("Shouldn't have an empty bag.") }
	if len(b.sub) > 1 {
		last := len(b.sub)-1
		if cb == b.crit[0] { return expanse(b.crit[1], b.crit[last]) }
		if cb == b.crit[last] { return expanse(b.crit[0], b.crit[last-1]) }
	} else if cb == b.crit[0] { return expanse0() }
	return b.expanse()
}
func (b *bag_t) debugPrint(prefix string) {
	fmt.Printf("%sbag -- prefix: %s, full: %v\n", prefix, b.key_, b.full)
	for i, t := range b.sub {
		fmt.Printf("%s cb: %d(%c) %T\n", prefix, b.crit[i], b.crit[i], t)
		t.debugPrint(prefix+"  ")
	}
}

/*
 span_t

 A span is a trie node that simply stores an array of sub-tries, where the critical byte
 for the sub-trie is index+start.  Range's are only used for sub-tries with high density.
*/
type span_t struct {
	key_ string
	val_ Value
	count_ int
	full bool
	start byte
	occupied_ uint16
	sub []itrie
}
func makeSpan(e expanse_t, key string, val Value, full bool) *span_t {
	s := new(span_t)
	s.key_ = key; s.val_ = val; s.full = full
	s.start = e.low
	s.sub = make([]itrie, e.size)
	return s
}
/*
 Constructs a new span with th contents of t and l, where l is always a leaf.  It is known
 that l starts a new sub-trie -- t does not have a sub-trie at critical byte cb.
*/
func span(t itrie, e expanse_t, cb byte, l *leaf_t) *span_t {
	s := makeSpan(e, t.key(), t.val(), t.hasVal())
	add := func(cb byte, t itrie) {	s.sub[cb - s.start] = t }
	t.withsubs(0, uint(cb), add)
	add(cb, l)
	t.withsubs(uint(cb+1), 256, add)
	s.count_ = t.count() + 1
	s.occupied_ = uint16(t.occupied() + 1)
	return s
}
/*
 Constructs a new span with the contents of t, minus the sub-trie at critical byte cb.  It
 is expected that any sub-trie at cb is a leaf.
*/
func spanWithout(t itrie, e expanse_t, without byte) *span_t {
	s := makeSpan(e, t.key(), t.val(), t.hasVal())
	add := func(cb byte, t itrie) { s.sub[cb - s.start] = t	}
	t.withsubs(uint(e.low), uint(without), add)
	t.withsubs(uint(without+1), uint(e.high)+1, add)
	s.count_ = t.count() - 1
	s.occupied_ = uint16(t.occupied() - 1)
	return s
}

func (s *span_t) clone() *span_t {
	n := new(span_t)
	*n = *s
	n.sub = make([]itrie, len(s.sub))
	copy(n.sub, s.sub)
	return n
}
func (s *span_t) cloneWithKey(key string) itrie {
	n := s.clone()
	n.key_ = str(key)
	return n
}
func (s *span_t) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := s.clone()
	n.key_ = str(key); n.val_ = val; n.full = true
	if !s.full { n.count_++; return n, 1 }
	return n, 0
}
func (s *span_t) with(e expanse_t, cb byte, key string, val Value) (itrie, int) {
	if e.low > s.start { panic("new start must be <= old start") }
	if int(e.size) < len(s.sub) { panic("new size must be >= old size") }
	n := new(span_t)
	*n = *s
	n.start = e.low
	n.sub = make([]itrie, int(e.size))
	copy(n.sub[s.start - n.start:], s.sub)
	i, added := int(cb - n.start), 0
	o := n.sub[i]
	n.sub[i], added = assoc(o, key, val)
	n.count_ += added
	if o == nil { n.occupied_++ }
	return n, added
}
func (s *span_t) assoc(key string, val Value) (itrie, int) {
	crit, match := findcrit(key, s.key_)
	if match {
		return s.cloneWithKeyValue(key, val)
	}

	prefix, cb, rest := splitKey(key, crit)
	_, _cb, _rest := splitKey(s.key_, crit)
	if crit < len(s.key_) {
		return bag2(prefix, nil, false, cb, _cb,
			leaf(rest, val), s.cloneWithKey(_rest)), 1
	}
	// Update expanse
	e0 := s.expanse()
	e := e0.with(cb)

	if e.size > e0.size {
		// Figure out if we're a span, a bag, or a bitmap.
		count := int(s.occupied_)+1
		if !spanOK(e, count) {
			// We're not a span.
			if count <= maxBagSize {
				return bag(s, cb, leaf(rest, val)), 1
			}
			// Prefer a bitmap
			return bitmap(s, cb, leaf(rest, val)), 1
		}
	}

	// Prefer a span -- the code below handles the case of adding a new child, or
	// overwriting an existing one.
	return s.with(e, cb, rest, val)
}
func (s *span_t) firstAfter(i int) byte {
	i++
	for ; i < len(s.sub); i++ {
		if s.sub[i] != nil { return byte(i) }
	}
	panic("no further occupied elements in span")
}
func (s *span_t) lastBefore(i int) byte {
	i--
	for ; i >= 0; i-- {
		if s.sub[i] != nil { return byte(i) }
	}
	panic("no prior occupied elements in span")
}
func (s *span_t) expanseWithout(cb byte) expanse_t {
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
func (s *span_t) without(key string) (itrie, int) {
	crit, match := findcrit(key, s.key_)
	if crit < len(s.key_) {
		// we don't have the element being removed
		return s, 0
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
		if ! s.full {
			// don't have the element
			return s, 0
		}
		n := s.clone()
		n.val_ = nil; n.full = false; n.count_--
		return n, 1
	}

	_, cb, rest := splitKey(key, crit)
	if cb < s.start {
		// we don't have the element being removed
		return s, 0
	}
	i := cb - s.start
	if int(i) >= len(s.sub) {
		// we don't have the element being removed
		return s, 0
	}
	n, less := s.sub[i].without(rest)
	if n == nil {
		// We removed a leaf -- shrink our children & possibly turn into a bag or leaf.
		occupied := s.occupied_ - 1
		// We shouldn't actually let spans get small enough to hit either of the next
		// two cases
		if occupied == 0 {
			if !s.full { panic("we should have a value if we have no sub-tries.") }
			return leaf(s.key_, s.val_), less
		} 
		if occupied == 1 && !s.full {
			o := 0
			for ; o < len(s.sub); o++ {
				if byte(o) != i && s.sub[o] != nil { break }
			}
			if o >= len(s.sub) { panic("We should have another valid sub-trie") }
			key = s.key_ + string(cb) + s.sub[o].key()
			return s.sub[o].cloneWithKey(key), less
		}
		e := s.expanse()
		if occupied >= minSpanSize {
			e = s.expanseWithout(cb)
			if spanOK(e, int(occupied)) {
				// We can stay a span
				return spanWithout(s, e, cb), less
			}
		}
		if occupied <= maxBagSize {
			// We should become a bag
			return bagWithout(s, e, cb), less
		}
		// Looks like we're a bitmap
		return bitmapWithout(s, e, cb), less
	}
	if less == 0 {
		if n != s.sub[i] { panic("Shouldn't make a new node without changes.") }
		return s, 0
	}
	s = s.clone()
	s.sub[i] = n; s.count_ -= less
	return s, less
}
func (s *span_t) entryAt(key string) itrie {
	crit, match := findcrit(key, s.key_)
	if match && s.full { return s }
	if crit >= len(key) { return nil }
	_, cb, rest := splitKey(key, crit)
	if cb >= s.start && int(cb) < (int(s.start)+len(s.sub)) {
		i := cb + s.start
		if s.sub[i] != nil { return s.sub[i].entryAt(rest) }
	}
	return nil
}
func (s *span_t) inorder(prefix string, ch chan Item) {
	root := len(prefix) == 0
	prefix += s.key_
	if s.full {
		ch <- Item{prefix, s.val_}
	}
	for i, b := range s.sub {
		if b != nil {
			b.inorder(prefix + string(s.start+byte(i)), ch)
		}
	}
	if root { close(ch) }
}
func (s *span_t) withsubs(start, end uint, f func(byte, itrie)) {
	start = uint(min(max(0, int(start) - int(s.start)), len(s.sub)))
	end = uint(min(max(0, int(end) - int(s.start)), len(s.sub)))
	if start >= end { return }
	for i, t := range s.sub[start:end] {
		if t == nil { continue }
		cb := s.start + byte(start) + byte(i)
		f(cb, t)
	}
}
func (s *span_t) key() string { return s.key_ }
func (s *span_t) hasVal() bool { return s.full }
func (s *span_t) val() Value { return s.val_ }
func (s *span_t) count() int { return s.count_ }
func (s *span_t) occupied() int { return int(s.occupied_) }
func (s *span_t) expanse() expanse_t { return expanse(s.start, s.start+byte(len(s.sub)-1)) }
func (s *span_t) debugPrint(prefix string) {
	fmt.Printf("%sspan -- prefix: %s, full: %v\n", prefix, s.key_, s.full)
	for i, t := range s.sub {
		if s.sub[i] == nil { continue }
		cb := s.start + byte(i)
		fmt.Printf("%s cb: %d(%c) %T\n", prefix, cb, cb, t)
		t.debugPrint(prefix + "  ")
	}
}

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
		n.sub[dst], added = b.sub[src].assoc(key, val); dst++; src++
	} else {
		n.sub[dst], added = assoc(nil, key, val); n.setbit(w, bit); dst++
	}
	copy(n.sub[dst:], b.sub[src:])
	n.count_ = b.count_ + added
	return n, added
}
func (b *bitmap_t) assoc(key string, val Value) (itrie, int) {
	crit, match := findcrit(key, b.key_)
	if match {
		return b.cloneWithKeyValue(key, val)
	}

	prefix, cb, rest := splitKey(key, crit)
	_, _cb, _rest := splitKey(b.key_, crit)
	if crit < len(b.key_) {
		return bag2(prefix, nil, false, cb, _cb,
			leaf(rest, val), b.cloneWithKey(_rest)), 1
	}

	// Figure out if we stay a bitmap or if we can become a span
	// we know we're too big to be a bag
	w, bit := bitpos(uint(cb))
	replace := b.isset(w, bit)
	if !replace {
		e := b.expanse().with(cb)
		count := len(b.sub)
		if spanOK(e, count+1) {
			// We can be a span
			return span(b, e, cb, leaf(rest, val)), 1
		}
	}
	// still a bitmap
	return b.with(cb, rest, val)
}
func (b *bitmap_t) without(key string) (itrie, int) {
	crit, match := findcrit(key, b.key_)
	if crit < len(b.key_) {
		// we don't have the element being removed
		return b, 0
	}

	if match {
		// we won't even check for the case of only 1 child in a bitmap -- it just
		// shouldn't happen
		n := b.clone()
		n.val_ = nil; n.full = false; n.count_--
		return n, 1
	}

	_, cb, rest := splitKey(key, crit)
	w, bit := bitpos(uint(cb))
	if !b.isset(w, bit) {
		// we don't have the element being removed
		return b, 0
	}
	i := b.indexOf(w, bit)
	n, less := b.sub[i].without(rest)
	if n == nil {
		// We removed a leaf -- shrink our children & possibly turn into a bag or span.
		occupied := b.occupied() - 1
		e := b.expanseWithout(cb)
		if spanOK(e, occupied) {
			// We can be a span
			return spanWithout(b, e, cb), less
		}
		if occupied <= maxBagSize {
			// We should become a bag
			return bagWithout(b, e, cb), less
		}
		// We should stay a bitmap
		return bitmapWithout(b, e, cb), less
	}
	if less == 0 {
		if n != b.sub[i] { panic("Shouldn't create a new node without changes.") }
		return b, 0
	}
	b = b.clone()
	b.sub[i] = n; b.clearbit(w, bit); b.count_--
	return b, less
}
func (b *bitmap_t) entryAt(key string) itrie {
	crit, match := findcrit(key, b.key_)
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
func (b *bitmap_t) inorder(prefix string, ch chan Item) {
	root := len(prefix) == 0
	prefix += b.key_
	if b.full {
		ch <- Item{prefix, b.val_}
	}
	b.withsubs(0, 256, func(cb byte, t itrie) {
		t.inorder(prefix + string(cb), ch)
	})
	if root { close(ch) }
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
func (b *bitmap_t) debugPrint(prefix string) {
	fmt.Printf("%sbitmap - prefix: %s, full: %v\n", prefix, b.key_, b.full)
	pr := func(cb byte, t itrie) {
		fmt.Printf("%s cb: %d(%c) %T\n", prefix, cb, cb, t)
		t.debugPrint(prefix + "  ")
	}
	b.withsubs(0, 256, pr)
}

