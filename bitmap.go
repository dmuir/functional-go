package immutable

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
	bm := new(bitmap_t); bm.key_ = str(key); bm.val_ = val; bm.full = full
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

