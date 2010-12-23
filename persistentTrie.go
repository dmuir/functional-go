package persistentMap

import (
	"fmt"
	"sort"
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

func expanse(first byte, rest ... byte) (byte, byte) {
	low := first
	high := first
	for _, v := range rest {
		if v < low { low = v }
		if v > high { high = v }
	}
	return high - low + 1, low
}

func spanOK(size byte, count int) bool {
	return int(size) <= (count + maxSpanWaste)
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


func assoc(m itrie, key string, val Value) (itrie, byte) {
	if m != nil {
		return m.assoc(key, val), 0
	}
	return leaf(key, val), 1
}

/*
 itrie.

 This is the interface of the internal trie nodes.
*/
type itrie interface {
	cloneWithKey(string) itrie
	cloneWithKeyValue(string, Value) itrie
	assoc(string, Value) itrie
	without(string) itrie
	entryAt(string) itrie
	count() int
	inorder(string, chan Item)
	withsubs(func (byte, itrie))
	k() string
	v() Value
}

/*
 trie.

 This struct implements the IPersistentMap interface via an internal itrie.
*/
type trie struct {
	n itrie
}
func (t *trie) Assoc(key string, val Value) IPersistentMap {
	n, _ := assoc(t.n, key, val)
	return &trie{n}
}
func (t *trie) Without(key string) IPersistentMap {
	if t.n != nil {
		return &trie{t.n.without(key)}
	}
	return nil
}
func (t *trie) Contains(key string) bool { 
	if t.n != nil {
		e := t.n.entryAt(key)
		return e != nil
	}
	return false
}
func (t *trie) ValueAt(key string) Value {
	if t.n != nil {
		e := t.n.entryAt(key)
		if e != nil { return e.v() }
	}
	panic(fmt.Sprintf("no value at: %s", key))
}
func (t *trie) Count() int {
	if t.n != nil {
		return t.n.count()
	}
	return 0
}
func (t *trie) Iter() chan Item {
	ch := make(chan Item)
	if t.n != nil {
		go t.n.inorder("", ch)
	} else {
		go close(ch)
	}
	return ch
}

/*
 leaf_t

 Internal node that contains only a key and a value.
*/
type leaf_t struct {
	key string
	val Value
}
func leaf(key string, val Value) *leaf_t {
	return &leaf_t{str(key), val}
}
func (l *leaf_t) clone() *leaf_t {
	return leaf(l.key, l.val)
}
func (l *leaf_t) cloneWithKey(key string) itrie {
	return leaf(key, l.val)
}
func (l *leaf_t) cloneWithKeyValue(key string, val Value) itrie {
	return leaf(key, val)
}
func (l *leaf_t) assoc(key string, val Value) itrie {
	crit, match := findcrit(key, l.key)
	if match {
		l.cloneWithKeyValue(key, val)
	}

	prefix, ch, rest := splitKey(key, crit)
	_, _ch, _rest := splitKey(l.key, crit)
	if crit == len(key) {
		return bag(prefix, val, true, critbytes(_ch), leaf(_rest, l.val))
	} else if crit == len(l.key) {
		return bag(prefix, l.val, true, critbytes(ch), leaf(rest, val))
	}
	return bag(prefix, nil, false, critbytes(ch, _ch), leaf(rest, val), leaf(_rest, l.val))
}
func (l *leaf_t) without(key string) itrie {
	if key == l.key { return nil }
	return l
}
func (l *leaf_t) entryAt(key string) itrie {
	if key == l.key { return l }
	return nil
}
func (l *leaf_t) count() int {
	return 1
}
func (l *leaf_t) inorder(prefix string, ch chan Item) {
	ch <- Item{prefix + l.key, l.val}
	if len(prefix) == 0 { close(ch) }
}
func (l *leaf_t) withsubs(func(byte, itrie)) {}
func (l *leaf_t) k() string { return l.key }
func (l *leaf_t) v() Value { return l.val }


/*
 bag_t

 A bag is an unordered collection of sub-tries.  It may hold a single value itself.
 When a bag has 4 or more elements, it may be promoted to a span.  If it has 8 or more elements,
 it will be promoted to a span or a bitmap.
*/
const (
	bagFull byte = 1 << iota
	bagSorted byte = 1 << iota
)

type bag_t struct {
	key string
	val Value
	flags byte
	crit [7]byte
	sub []itrie
}
func (b *bag_t) full() bool { return b.flags & bagFull != 0 }
func (b *bag_t) sorted() bool { return b.flags & bagSorted != 0 }
func bag(key string, val Value, full bool, crit string, sub ... itrie) itrie {
	b := new(bag_t)
	b.key = str(key); b.val = val
	if full { b.flags |= bagFull }
	copy(b.crit[:], crit)
	b.sub = make([]itrie, len(sub))
	copy(b.sub, sub)
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
	n.key = str(key)
	return n
}
func (b *bag_t) cloneWithKeyValue(key string, val Value) itrie {
	n := b.clone()
	n.key = str(key)
	n.val = val
	return n
}
func (b *bag_t) with(i int, ch byte, key string, val Value) itrie {
	n := new(bag_t)
	*n = *b
	if i >= len(b.sub) {
		n.sub = make([]itrie, i+1)
	} else {
		n.sub = make([]itrie, len(b.sub))
	}
	n.crit[i] = ch
	copy(n.sub, b.sub)
	n.sub[i], _ = assoc(n.sub[i], key, val)
	return n
}
func (b *bag_t) find(ch byte) int {
	// don't bother sorting -- we know there's only a handful of items.
	// Linear search will suffice.
	for i := 0; i < len(b.sub); i++ {
		if ch == b.crit[i] {
			return i
		}
	}
	return len(b.sub)
}
func (b *bag_t) sort() {
	// Even though we're an immutable data structure, we can sort the bag in-place
	// since it won't change the behavior with respect to the public interface.
	if !b.sorted() {
		sorter := sortBag{b.crit[:len(b.sub)], b.sub}
		sort.Sort(sorter)
	}
	b.flags |= bagSorted
}

func (b *bag_t) assoc(key string, val Value) itrie {
	crit, match := findcrit(key, b.key)
	if match {
		return b.cloneWithKeyValue(key, val)
	}
	
	prefix, ch, rest := splitKey(key, crit)
	_, _ch, _rest := splitKey(b.key, crit)	
	if crit < len(b.key) {
		return bag(prefix, nil, false, critbytes(ch, _ch),
			leaf(rest, val), b.cloneWithKey(_rest))
	}
	i := b.find(ch)
	if i >= len(b.sub) {
		// Determine whether we're a bag, span, or a bitmap
		if i >= minSpanSize {
			size, start := expanse(ch, b.crit[0:len(b.sub)]...)
			if spanOK(size, i) {
				// Prefer a span, even if we're small enough to stay bag
				sub := make([]itrie, int(size))
				for i := 0; i < len(b.sub); i++ {
					sub[b.crit[i] - start] = b.sub[i]
				}
				sub[ch-start] = leaf(rest, val)
				return span(b.key, b.val, b.full(), start, byte(i+1), sub)
			}
		}
		if i >= maxBagSize {
			// Prefer a bitmap
			return bitmapFromBag(b, ch, rest, val)
		}
	}
		
	// We still fit in a bag
	return b.with(i, ch, rest, val)
}

func (b *bag_t) without(key string) itrie {
	crit, match := findcrit(key, b.key)
	if crit < len(b.key) {
		// we don't have the element being removed
		return b
	}

	if match {
		if len(b.sub) == 1 {
			// collapse this node to it's only child.
			key += string(b.crit[0]) + b.sub[0].k()
			return b.sub[0].cloneWithKey(key)
		}
		n := b.clone()
		n.val = nil
		n.flags &= ^bagFull
		return n
	}

	_, ch, rest := splitKey(key, crit)
	i := b.find(ch)
	if i >= len(b.sub) {
		// we don't have the element being removed
		return b
	}
	n := b.sub[i].without(rest)
	if n == nil {
		// We removed a leaf -- shrink our children & possibly turn into a leaf.
		last := len(b.sub)-1
		if i < last {
			// We're going to move the subtrie we're "removing" to the end of 
			// the slice.  Even though these are "immutable" data structures,
			// we can do this since we're not changing the externally visible
			// behavior
			b.sub[i], b.sub[last] = b.sub[last], b.sub[i]
			b.crit[i], b.crit[last] = b.crit[last], b.crit[i]
			b.flags &= ^bagSorted
			i = last
		}
		if last == 0 {
			if !b.full() {
				panic("we should have a value if we have no sub-tries.")
			}
			return leaf(b.key, b.val)
		} else if last == 1 && !b.full() {
			key = b.key + string(b.crit[0]) + b.sub[0].k()
			return b.sub[0].cloneWithKey(key)
		}
		if last >= minSpanSize {
			size, start := expanse(b.crit[0], b.crit[1:last]...)
			if spanOK(size, last) {
				// We can be a range
				sub := make([]itrie, int(size))
				b.withsubs(func(cb byte, t itrie) {
					sub[cb - start] = t
				})
				return span(b.key, b.val, b.full(), start, byte(last), sub)
			}
		}
		// Still a bag.
		sub := make([]itrie, last)
		copy(sub, b.sub[:last])
		crit := make([]byte, last)
		copy(crit, b.crit[:last])
		return bag(b.key, b.val, b.full(), string(crit), sub...)
	}
	b = b.clone()
	b.sub[i] = n
	return b
}
func (b *bag_t) entryAt(key string) itrie {
	crit, match := findcrit(key, b.key)
	if match && b.full() { return b }
	if crit >= len(key) { return nil }
	_, ch, rest := splitKey(key, crit)
	i := b.find(ch)
	if i >= len(b.sub) { return nil }
	return b.sub[i].entryAt(rest)
}
func (b *bag_t) count() int {
	count := 0
	if b.full() { count++ }
	for _, c := range b.sub {
		count += c.count()
	}
	return count
}
func (b *bag_t) inorder(prefix string, ch chan Item) {
	root := len(prefix) == 0
	prefix += b.key
	if b.full() {
		ch <- Item{prefix, b.val}
	}
	if !b.sorted() { b.sort() }
	for i := 0; i < len(b.sub); i++ {
		b.sub[i].inorder(prefix + string(b.crit[i]), ch)
	}
	if root { close(ch) }
}
func (b *bag_t) withsubs(f func(byte, itrie)) {
	if !b.sorted() { b.sort() }
	for i := 0; i < len(b.sub); i++ {
		f(b.crit[i], b.sub[i])
	}
}
func (b *bag_t) k() string { return b.key }
func (b *bag_t) v() Value { return b.val }



/*
 sortBag

 Implements sort.Interface for bag_t's.
*/
type sortBag struct {
	crit []byte
	sub []itrie
}
func (n sortBag) Len() int {
	return len(n.sub)
}
func (n sortBag) Less(i, j int) bool {
	return n.crit[i] < n.crit[j]
}
func (n sortBag) Swap(i, j int) {
	n.crit[i], n.crit[j] = n.crit[j], n.crit[i]
	n.sub[i], n.sub[j] = n.sub[j], n.sub[i]
}

/*
 span_t

 A span is a trie node that simply stores an array of sub-tries, where the critical byte
 for the sub-trie is index+start.  Range's are only used for sub-tries with high density.
*/
type span_t struct {
	key string
	val Value
	full bool
	start byte
	occupied byte
	sub []itrie
}
func span(key string, val Value, full bool, start byte, occupied byte, sub []itrie) itrie {
	return &span_t{str(key), val, full, start, occupied, sub}
}
func (r *span_t) clone() *span_t {
	n := new(span_t)
	*n = *r
	n.sub = make([]itrie, len(r.sub))
	copy(n.sub, r.sub)
	return n
}
func (r *span_t) cloneWithKey(key string) itrie {
	n := r.clone()
	n.key = str(key)
	return n
}
func (r *span_t) cloneWithKeyValue(key string, val Value) itrie {
	n := r.clone()
	n.key = str(key)
	n.val = val
	return n
}
func (s *span_t) assoc(key string, val Value) itrie {
	crit, match := findcrit(key, s.key)
	if match {
		return s.cloneWithKeyValue(key, val)
	}

	prefix, ch, rest := splitKey(key, crit)
	_, _ch, _rest := splitKey(s.key, crit)
	if crit < len(s.key) {
		return bag(prefix, nil, false, critbytes(ch, _ch),
			leaf(rest, val), s.cloneWithKey(_rest))
	}
	// Update expanse
	oldsize := byte(len(s.sub))
	size, start := expanse(ch, s.start, s.start + oldsize-1)
	
	if size > oldsize {
		// Figure out if we're a span, a bag, or a bitmap.
		count := int(s.occupied)+1
		if spanOK(size, count) {
			// We're not a span.
			if count <= maxBagSize {
				// Prefer a bag
				crits := make([]byte, count)
				crits[0] = ch
				sub := make([]itrie, count)
				sub[0] = leaf(rest, val)
				next := 1
				for i, child := range s.sub {
					if child != nil {
						crits[next] = s.start + byte(i)
						sub[next] = child
						next++
					}
				}
				return bag(s.key, s.val, s.full, string(crits), sub...)
			}
			// Prefer a bitmap
			return bitmapFromSpan(s, ch, rest, val)
		}
	}

	// Prefer a span -- the code below handles the case of adding a new child, or
	// overwriting an existing one.
	if start > s.start { panic("new start must be <= old start") }
	if size < oldsize { panic("new size must be >= old size") }
	sub := make([]itrie, int(size))
	copy(sub[s.start - start:], s.sub)
	child, added := assoc(sub[ch-start], rest, val)
	sub[ch - start] = child
	return span(s.key, s.val, s.full, start, s.occupied+added, sub)
}
func (s *span_t) firstAfter(i byte) byte {
	i++
	for ; int(i) < len(s.sub); i++ {
		if s.sub[i] != nil { return i }
	}
	panic("no further occupied elements in span")
}
func (s *span_t) lastBefore(i byte) byte {
	i--
	for ; i >= 0; i-- {
		if s.sub[i] != nil { return i }
	}
	panic("no prior occupied elements in span")
}
func (s *span_t) expanseWithout(ch byte) (byte, byte) {
	var low, high byte = s.start, s.start + byte(len(s.sub)-1)
	if ch == low {
		i := s.firstAfter(0)
		return high - low - i + byte(1), low + byte(i)
	}
	if ch == high {
		i := s.lastBefore(high)
		return i + 1, low
	}
	return high - low + 1, low
}
func (s *span_t) without(key string) itrie {
	crit, match := findcrit(key, s.key)
	if crit < len(s.key) {
		// we don't have the element being removed
		return s
	}

	if match {
		if s.occupied == 1 {
			for i, c := range s.sub {
				// collapse this node to it's only child.
				if c == nil { continue }
				key += string(s.start+byte(i)) + c.k()
				return c.cloneWithKey(key)
			}
			panic("should have found a non-nil sub-trie.")
		}
		n := s.clone()
		n.val = nil
		n.full = false
		return n
	}

	_, ch, rest := splitKey(key, crit)
	if ch < s.start {
		// we don't have the element being removed
		return s
	}
	i := ch - s.start
	if int(i) >= len(s.sub) {
		// we don't have the element being removed
		return s
	}
	n := s.sub[i].without(rest)
	if n == nil {
		// We removed a leaf -- shrink our children & possibly turn into a bag or leaf.
		occupied := s.occupied - 1
		// We shouldn't actually let spans get small enough to hit either of the next
		// two cases
		if occupied == 0 {
			if !s.full { panic("we should have a value if we have no sub-tries.") }
			return leaf(s.key, s.val)
		} 
		if occupied == 1 && !s.full {
			o := 0
			for ; o < len(s.sub); o++ {
				if byte(o) != i && s.sub[o] != nil { break }
			}
			if o >= len(s.sub) { panic("We should have another valid sub-trie") }
			key = s.key + string(ch) + s.sub[o].k()
			return s.sub[o].cloneWithKey(key)
		}
		if occupied >= minSpanSize {
			// We can stay a span
			size, start := s.expanseWithout(ch)
			if spanOK(size, int(occupied)) {
				sub := make([]itrie, occupied)
				if start < s.start {
					panic("when shrinking a span, start can only grow.")
				}
				offset := start - s.start
				copy(sub[0:i], s.sub[offset:i])
				copy(sub[i:], s.sub[offset+i+1:])
				return span(s.key, s.val, s.full, start, occupied, sub)
			}
		}
		// We should become a bag
		sub := make([]itrie, occupied)
		crit := make([]byte, occupied)
		split := i
		index := 0
		add := func (i int, c itrie) {
			if s.sub[i] == nil { return }
			crit[index] = s.start + byte(i)
			sub[index] = s.sub[i]
			index++
		}
		for i, c := range s.sub[0:split] { add(i, c) }
		for i, c := range s.sub[split+1:] { add(i, c) }
		return bag(s.key, s.val, s.full, string(crit), sub...)
	}
	s = s.clone()
	s.sub[i] = n
	return s
}
func (s *span_t) entryAt(key string) itrie {
	crit, match := findcrit(key, s.key)
	if match && s.full { return s }
	if crit >= len(key) { return nil }
	_, ch, rest := splitKey(key, crit)
	if ch >= s.start && int(ch) < (int(s.start)+len(s.sub)) {
		i := ch + s.start
		if s.sub[i] != nil { return s.sub[i].entryAt(rest) }
	}
	return nil
}
func (s *span_t) count() int {
	count := 0
	if s.full { count++ }
	for _, c := range s.sub {
		if c != nil { count += c.count() }
	}
	return count
}
func (s *span_t) inorder(prefix string, ch chan Item) {
	root := len(prefix) == 0
	prefix += s.key
	if s.full {
		ch <- Item{prefix, s.val}
	}
	for i, b := range s.sub {
		if b != nil {
			b.inorder(prefix + string(s.start+byte(i)), ch)
		}
	}
	if root { close(ch) }
}
func (s *span_t) withsubs(f func(byte, itrie)) {
	for i, c := range s.sub {
		if c == nil { continue }
		cb := s.start + byte(i)
		f(cb, c)
	}
}
func (s *span_t) k() string { return s.key }
func (s *span_t) v() Value { return s.val }


/*
 bitmap_t

 A bitmap is a trie node that uses a bitmap to track which of it's children are occupied.
*/
type bitmap_t struct {
	key string
	val Value
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
func (b *bitmap_t) bitpos(ch byte) (byte, uint64) {
	return (ch >> 6), uint64(1) << (ch & 0x3f)
}
func (b *bitmap_t) setbit(w byte, bit uint64) {
	b.bm[w] |= bit
	for ; w < 3; w++ { b.off[w+1] += 1 }
}
func (b *bitmap_t) clearbit(w byte, bit uint64) {
	b.bm[w] &= ^bit
	for ; w < 3; w++ { b.off[w+1] -= 1 }
}
func (b *bitmap_t) isset(w byte, bit uint64) bool {
	return b.bm[w] & bit != 0
}
func (b *bitmap_t) setbits(first byte, rest ... byte) {
	var off [4]uint8
	w, bit := b.bitpos(first)
	b.bm[w] |= bit
	off[w] += 1
	for _, p := range rest {
		w, bit := b.bitpos(p)
		b.bm[w] |= bit
		off[w] += 1
	}
	off[3] = off[2] + off[1] + off[0]
	off[2] = off[1] + off[0]
	off[1] = off[0]
	off[0] = 0
	b.off = off
}
func (b *bitmap_t) indexOf(w byte, bit uint64) byte {
	return countbits(b.bm[w] & (bit-1)) + b.off[w]
}
func (b *bitmap_t) min() byte {
	for w, bm := range b.bm {
		if bm == 0 { continue }
		bit := bm ^ (bm & (bm-1))
		return countbits(bit-1) + byte(w*64)
	}
	panic("Didn't find any bits set in bitmap")
}
func (b *bitmap_t) max() byte {
	for w := 3; w >= 0; w-- {
		if b.bm[w] == 0 { continue }
		bm := reverse(b.bm[w])
		bit := bm ^ (bm & (bm-1))
		return (byte(63) - countbits(bit-1)) + byte(w*64)
	}
	panic("Didn't find any bits set in bitmap")
}
func (b *bitmap_t) minmax() (byte, byte) {
	return b.min(), b.max()
}
func (b *bitmap_t) grow(num int) *bitmap_t {
	n := new(bitmap_t)
	*n = *b
	n.sub = make([]itrie, num)
	copy(n.sub, b.sub)
	return n
}
func bitmapFromBag(b *bag_t, ch byte, key string, val Value) itrie {
	bm := new(bitmap_t)
	bm.key = b.key
	bm.val = b.val
	bm.full = b.full()
	bm.sub = make([]itrie, len(b.sub)+1)

	// We sort the bag so that we can do everything in one pass.
	b.sort()
	// Find where we're inserting the new leaf
	ins := func(ch byte, bytes []byte) int {
		for i, c := range bytes {
			if c == ch { panic("matching crit bytes shouldn't be possible") }
			if ch < c { return i }
		}
		return len(bytes)
	} (ch, b.crit[0:len(b.sub)])
	// Now we can run everything sequentially
	copy(bm.sub[0:ins], b.sub[0:ins])
	bm.sub[ins] = leaf(key, val)
	copy(bm.sub[ins+1:], b.sub[ins:])

	// Initialize the bitmap
	bm.setbits(ch, b.crit[0:len(b.sub)]...)
	return bm
}
func bitmapFromSpan(s *span_t, ch byte, key string, val Value) itrie {
	b := new(bitmap_t)
	b.key = s.key
	b.val = s.val
	b.full = s.full
	b.sub = make([]itrie, s.occupied+1)
	crits := make([]byte, s.occupied)

	// We know that ch falls outside of the span (otherwise we wouldn't be here)
	index := 0
	if ch < s.start {
		// The new leaf gets the first index.
		b.sub[index] = leaf(key, val)
	}
	critindex := 0
	for i, child := range s.sub {
		crits[critindex] = byte(i)+s.start
		critindex++
		b.sub[index] = child
		index++
	}
	b.setbits(ch, crits...)
	if index < len(b.sub) {
		// New branch comes after the span -- it gets the last index
		b.sub[index] = leaf(key, val)
	}
	return b
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
	n.key = str(key)
	return n
}
func (b *bitmap_t) cloneWithKeyValue(key string, val Value) itrie {
	n := b.clone()
	n.key = str(key)
	n.val = val
	return n
}

func (b *bitmap_t) assoc(key string, val Value) itrie {
	crit, match := findcrit(key, b.key)
	if match {
		return b.cloneWithKeyValue(key, val)
	}

	prefix, ch, rest := splitKey(key, crit)
	_, _ch, _rest := splitKey(b.key, crit)
	if crit < len(b.key) {
		return bag(prefix, nil, false, critbytes(ch, _ch),
			leaf(rest, val), b.cloneWithKey(_rest))
	}

	// Figure out if we stay a bitmap or if we can become a span
	// we know we're too big to be a bag
	size, start := expanse(b.min(), b.max(), ch)
	w, bit := b.bitpos(ch)
	count := len(b.sub)
	replace := b.isset(w, bit)
	if !replace { count++ }
	if replace || !spanOK(size, count) {
		// We stay a bitmap
		n := new(bitmap_t)
		*n = *b
		n.sub = make([]itrie, count)
		// determine the index for the branch we're adding/replacing
		index := n.indexOf(w, bit)
		// copy branches preceeding the branch we're adding/replacing
		copy(b.sub[0:index], n.sub[0:index])
		if !replace {
			n.sub[index] = leaf(rest, val)
			n.setbit(w, bit)
			copy(n.sub[index+1:], b.sub[index:])
		} else {
			n.sub[index] = b.sub[index].assoc(rest, val)
			copy(n.sub[index+1:], b.sub[index+1:])
		}
		return n
	}

	// We can be a span
	sub := make([]itrie, int(size))
	b.withsubs(func(cb byte, t itrie) {
		sub[cb - start] = t
	})
	sub[ch-start] = leaf(rest, val)
	return span(b.key, b.val, b.full, start, byte(count+1), sub)
}
func (b *bitmap_t) without(key string) itrie {
	crit, match := findcrit(key, b.key)
	if crit < len(b.key) {
		// we don't have the element being removed
		return b
	}

	if match {
		// we won't even check for the case of only 1 child in a bitmap -- it just
		// shouldn't happen
		n := b.clone()
		n.val = nil
		n.full = false
		return n
	}

	_, ch, rest := splitKey(key, crit)
	w, bit := b.bitpos(ch)
	if !b.isset(w, bit) {
		// we don't have the element being removed
		return b
	}
	i := b.indexOf(w, bit)
	n := b.sub[i].without(rest)
	if n == nil {
		// We removed a leaf -- shrink our children & possibly turn into a bag or span.
		occupied := len(b.sub) - 1
		size, start := expanse(b.min(), b.max(), ch)
		if spanOK(size, occupied) {
			// We can be a span
			sub := make([]itrie, int(size))
			b.withsubs(func(cb byte, t itrie) {
				sub[cb - start] = t
			})
			return span(b.key, b.val, b.full, start, byte(occupied), sub)
		}
		if occupied <= maxBagSize {
			// We should become a bag
			sub := make([]itrie, occupied)
			crit := make([]byte, occupied)
			i := 0
			b.withsubs(func(cb byte, t itrie) {
				crit[i] = cb; sub[i] = t; i++
			})
			return bag(b.key, b.val, b.full, string(crit), sub...)
		}
		// We should stay a bitmap
		n := b.clone()
		n.sub = make([]itrie, occupied)
		copy(n.sub[0:i], b.sub[0:i])
		copy(n.sub[i:], b.sub[i+1:])
		n.clearbit(w, bit)
		return n
	}
	b = b.clone()
	b.sub[i] = n
	b.clearbit(w, bit)
	return b
}
func (b *bitmap_t) entryAt(key string) itrie {
	crit, match := findcrit(key, b.key)
	if match && b.full { return b }
	if crit >= len(key) { return nil }
	_, ch, rest := splitKey(key, crit)
	w, bit := b.bitpos(ch)
	if b.isset(w, bit) {
		index := b.indexOf(w, bit)
		return b.sub[index].entryAt(rest)
	}
	return nil
}
func (b *bitmap_t) count() int {
	count := 0
	if b.full { count++ }
	for _, c := range b.sub {
		count += c.count()
	}
	return count
}
func (b *bitmap_t) inorder(prefix string, ch chan Item) {
	root := len(prefix) == 0
	prefix += b.key
	if b.full {
		ch <- Item{prefix, b.val}
	}
	b.withsubs(func(cb byte, t itrie) {
		t.inorder(prefix + string(cb), ch)
	})
	if root { close(ch) }
}
func (b *bitmap_t) withsubs(f func(byte, itrie)) {
	i := 0
	for w, bm := range b.bm {
		for bm != 0 {
			bit := bm ^ (bm & (bm - 1))
			cb := countbits(bit-1) + byte(64*w)
			f(cb, b.sub[i])
			bm &= (bm-1)
			i++
		}
	}
}
func (b *bitmap_t) k() string { return b.key }
func (b *bitmap_t) v() Value { return b.val }

func NewTrie() IPersistentMap {
	return new(trie)
}


