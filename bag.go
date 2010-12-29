package immutable

import "fmt"

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

