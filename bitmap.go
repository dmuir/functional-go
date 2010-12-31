package immutable

/*
 bitmap_t

 A bitmap is a trie node that uses a bitmap to track which of it's children are occupied.
*/
type bitmap_ struct {
	entry_
	off [4]uint8
	count_ int
	bm [4]uint64
	sub []itrie
}
type bitmapK struct {
	entryK
	bitmap_
}
type bitmapV struct {
	entryV
	bitmap_
}
type bitmapKV struct {
	entryKV
	bitmap_
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
func (b *bitmap_) setbit(w int, bit uint64) {
	b.bm[w] |= bit
	for ; w < 3; w++ { b.off[w+1] += 1 }
}
func (b *bitmap_) clearbit(w int, bit uint64) {
	b.bm[w] &= ^bit
	for ; w < 3; w++ { b.off[w+1] -= 1 }
}
func (b *bitmap_) isset(w int, bit uint64) bool {
	return b.bm[w] & bit != 0
}
func (b *bitmap_) indexOf(w int, bit uint64) int {
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
func (b *bitmap_) min() byte {
	for w, bm := range b.bm {
		if bm == 0 { continue }
		return minbit(bm) + byte(64*w)
	}
	panic("Didn't find any bits set in bitmap")
}
func (b *bitmap_) max() byte {
	for w := 3; w >= 0; w-- {
		if b.bm[w] == 0 { continue }
		return maxbit(b.bm[w]) + byte(64*w)
	}
	panic("Didn't find any bits set in bitmap")
}
func (b *bitmap_) lastBefore(cb byte) byte {
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
func (b *bitmap_) firstAfter(cb byte) byte {
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
func makeBitmap(size int, key string, val Value, full bool) (b *bitmap_, t itrie) {
	if len(key) > 0 {
		if full {
			Cumulative[kBitmapKV]++
			n := new(bitmapKV)
			n.key_ = str(key); n.val_ = val
			b, t = &n.bitmap_, n
		} else {
			Cumulative[kBitmapK]++
			n := new(bitmapK)
			n.key_ = str(key)
			b, t = &n.bitmap_, n
		}
	} else {
		if full {
			Cumulative[kBitmapV]++
			n := new(bitmapV)
			n.val_ = val
			b, t = &n.bitmap_, n
		} else {
			Cumulative[kBitmap_]++
			n := new(bitmap_)
			b, t = n, n
		}
	}
	b.sub = make([]itrie, int(size))
	return
}
/*
 Constructs a new bitmap with the contents of t and l, where l is always a leaf.  It is known
 that l starts a new sub-trie -- t does not have a sub-trie at critical byte cb.
*/
func bitmap(t itrie, cb byte, l itrie) itrie {
	bm, r := makeBitmap(t.occupied()+1, t.key(), t.val(), t.hasVal())
	index := 0
	add := func(cb byte, t itrie) {
		w, bit := bitpos(uint(cb))
		bm.sub[index] = t; bm.setbit(w, bit); index++
	}
	t.withsubs(0, uint(cb), add)
	add(cb, l)
	t.withsubs(uint(cb+1), 256, add)
	bm.count_ = t.count() + 1
	return r
}
/*
 Constructs a new bitmap with the contents of t, minus the sub-trie at critical byte cb.  It
 is expected that any sub-trie at cb is a leaf.
*/
func bitmapWithout(t itrie, e expanse_t, without byte) itrie {
	bm, r := makeBitmap(t.occupied()-1, t.key(), t.val(), t.hasVal())
	index := 0
	add := func(cb byte, t itrie) { 
		bm.sub[index] = t; bm.setbit(bitpos(uint(cb))); index++
	}
	t.withsubs(uint(e.low), uint(without), add)
	t.withsubs(uint(without+1), uint(e.high)+1, add)
	bm.count_ = t.count() - 1
	return r
}

func (b *bitmap_) copy(t *bitmap_) {
	b.count_ = t.count_
	b.off = t.off
	b.bm = t.bm
	b.sub = make([]itrie, len(t.sub))
	copy(b.sub, t.sub)
}	
func (b *bitmap_) cloneWithKey(key string) itrie {
	n := new(bitmapK)
	Cumulative[kBitmapK]++
	n.bitmap_ = *b; n.key_ = str(key)
	return n
}
func (b *bitmapV) cloneWithKey(key string) itrie {
	n := new(bitmapKV)
	Cumulative[kBitmapKV]++
	n.bitmap_ = b.bitmap_; n.key_ = str(key); n.val_ = b.val_
	return n
}
func (b *bitmapKV) cloneWithKey(key string) itrie {
	n := new(bitmapKV)
	Cumulative[kBitmapKV]++
	n.bitmap_ = b.bitmap_; n.key_ = str(key); n.val_ = b.val_
	return n
}
func (b *bitmap_) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := new(bitmapKV)
	Cumulative[kBitmapKV]++
	n.bitmap_ = *b; n.key_ = str(key); n.val_ = val; n.count_++
	return n, 1
}
func (b *bitmapV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := new(bitmapKV)
	Cumulative[kBitmapKV]++
	n.bitmap_ = b.bitmap_; n.key_ = str(key); n.val_ = val
	return n, 0
}
func (b *bitmapKV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := new(bitmapKV)
	Cumulative[kBitmapKV]++
	n.bitmap_ = b.bitmap_; n.key_ = str(key); n.val_ = val
	return n, 0
}
func (b *bitmap_) modify(incr, i int, sub itrie) itrie {
	n := new(bitmap_)
	n.copy(b); n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bitmapK) modify(incr, i int, sub itrie) itrie {
	n := new(bitmapK)
	n.copy(&b.bitmap_); n.key_ = b.key_; n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bitmapV) modify(incr, i int, sub itrie) itrie {
	n := new(bitmapV)
	n.copy(&b.bitmap_); n.val_ = b.val_; n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bitmapKV) modify(incr, i int, sub itrie) itrie {
	n := new(bitmapKV)
	n.copy(&b.bitmap_); n.key_ = b.key_; n.val_ = b.val_; n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bitmap_) withoutValue() (itrie, int) {
	return b, 0
}
func (b *bitmapV) withoutValue() (itrie, int) {
	n := b.bitmap_
	return &n, 1
}
func (b *bitmapKV) withoutValue() (itrie, int) {
	n := new(bitmapK)
	n.bitmap_ = b.bitmap_; n.key_ = b.key_
	return n, 1
}
func (n *bitmap_) withBitmap(b *bitmap_, incr int, cb byte, r itrie) {
	*n = *b; n.count_ = b.count_ + incr
	w, bit := bitpos(uint(cb))
	size := len(b.sub)
	exists := b.isset(w, bit)
	if !exists { size++ }
	i := b.indexOf(w, bit)
	n.sub = make([]itrie, size)
	copy(n.sub[:i], b.sub[:i])
	src, dst := i, i
	n.sub[dst] = r; dst++; if exists { src++ } else { n.setbit(w, bit) }
	copy(n.sub[dst:], b.sub[src:])
}
func (b *bitmap_) with(incr int, cb byte, r itrie) itrie {
	t := b.maybeGrow(b, cb, r)
	if t != nil { return t }
	n := new(bitmap_)
	Cumulative[kBitmap_]++
	n.withBitmap(b, incr, cb, r)
	return n
}
func (b *bitmapK) with(incr int, cb byte, r itrie) itrie {
	t := b.maybeGrow(b, cb, r)
	if t != nil { return t }
	n := new(bitmapK)
	Cumulative[kBitmap_]++
	n.key_ = b.key_
	n.withBitmap(&b.bitmap_, incr, cb, r)
	return n
}
func (b *bitmapV) with(incr int, cb byte, r itrie) itrie {
	t := b.maybeGrow(b, cb, r)
	if t != nil { return t }
	n := new(bitmapV)
	Cumulative[kBitmap_]++
	n.val_ = b.val_
	n.withBitmap(&b.bitmap_, incr, cb, r)
	return n
}
func (b *bitmapKV) with(incr int, cb byte, r itrie) itrie {
	t := b.maybeGrow(b, cb, r)
	if t != nil { return t }
	n := new(bitmapKV)
	Cumulative[kBitmap_]++
	n.key_ = b.key_; n.val_ = b.val_
	n.withBitmap(&b.bitmap_, incr, cb, r)
	return n
}
func (b *bitmap_) subAt(cb byte) itrie {
	w, bit := bitpos(uint(cb))
	if !b.isset(w, bit) { return nil }
	return b.sub[b.indexOf(w, bit)]
}
func (b *bitmap_) maybeGrow(t itrie, cb byte, r itrie) itrie {
	// Figure out if we stay a bitmap or if we can become a span
	// we know we're too big to be a bag
	w, bit := bitpos(uint(cb))
	exists := b.isset(w, bit)
	if !exists {
		e := b.expanse().with(cb)
		count := len(b.sub)
		if spanOK(e, count+1) {
			// We can be a span
			return span(t, e, cb, r)
		}
	}
	// still a bitmap
	return nil
}
func (b *bitmap_) without(t itrie, key string) (itrie, int) {
	return b.without_(t, key, 0)
}
func (b *bitmapK) without(t itrie, key string) (itrie, int) {
	crit, _ := findcb(key, b.key_)
	if crit < len(b.key_) {
		return t, 0
	}
	return b.bitmap_.without_(t, key, crit)
}
func (b *bitmapV) without(t itrie, key string) (itrie, int) {
	if len(key) == 0 {
		// we won't even check for the case of only 1 child in a bitmap -- it just
		// shouldn't happen
		return t.withoutValue()
	}
	return b.bitmap_.without_(t, key, 0)
}
func (b *bitmapKV) without(t itrie, key string) (itrie, int) {
	crit, match := findcb(key, b.key_)
	if crit < len(b.key_) {
		return t, 0
	}
	if match {
		// we won't even check for the case of only 1 child in a bitmap -- it just
		// shouldn't happen
		return t.withoutValue()
	}
	return b.bitmap_.without_(t, key, crit)
}
func (b *bitmap_) without_(t itrie, key string, crit int) (itrie, int) {
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
func (b *bitmapKV) foreach(prefix string, f func(string, Value)) {
	prefix += b.key_
	f(prefix, b.val_)
	b.bitmap_.foreach(prefix, f)
}
func (b *bitmapV) foreach(prefix string, f func(string, Value)) {
	f(prefix, b.val_)
	b.bitmap_.foreach(prefix, f)
}
func (b *bitmapK) foreach(prefix string, f func(string, Value)) {
	prefix += b.key_
	b.bitmap_.foreach(prefix, f)
}
func (b *bitmap_) foreach(prefix string, f func(string, Value)) {
	b.withsubs(0, 256, func(cb byte, t itrie) {
		t.foreach(prefix + string(cb), f)
	})
}
func (b *bitmap_) withsubs(start, end uint, f func(byte, itrie)) {
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
func (b *bitmap_) count() int { return b.count_ }
func (b *bitmap_) occupied() int { return len(b.sub) }
func (b *bitmap_) expanse() expanse_t { return expanse(b.min(), b.max()) }
func (b *bitmap_) expanseWithout(cb byte) expanse_t {
	e := b.expanse()
	if cb == e.low {
		e.low = b.firstAfter(cb)
	}
	if cb == e.high {
		e.high = b.lastBefore(cb)
	}
	return expanse(e.low, e.high)
	
}

