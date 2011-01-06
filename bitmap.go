package immutable

import "unsafe"
import "runtime"

/*
 bitmap_t

 A bitmap is a trie node that uses a bitmap to track which of it's children are occupied.
*/
type bitmap_ struct {
	entry_
	occupied_ uint16
	count_ int
	off [4]uint8
	bm [4]uint64
	sub [256]itrie		// We don't actually allocate 256 entries
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

var sizeofBitmap_ uintptr
var sizeofBitmapK uintptr
var sizeofBitmapV uintptr
var sizeofBitmapKV uintptr
var sizeofSub uintptr

func init() {
	var b_ bitmap_
	var bk bitmapK
	var bv bitmapV
	var bkv bitmapKV
	var t itrie

	sizeofSub = uintptr(unsafe.Sizeof(t))
	sizeofBitmap_ = uintptr(unsafe.Sizeof(b_)) - 256*sizeofSub
	sizeofBitmapK = uintptr(unsafe.Sizeof(bk)) - 256*sizeofSub
	sizeofBitmapV = uintptr(unsafe.Sizeof(bv)) - 256*sizeofSub
	sizeofBitmapKV = uintptr(unsafe.Sizeof(bkv)) - 256*sizeofSub
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
func newBitmap_(size uint16) *bitmap_ {
	asize := sizeofSub*uintptr(size)+sizeofBitmap_
	b := (*bitmap_)(unsafe.Pointer(runtime.Alloc(asize)))
	b.occupied_ = size
	return b
}
func newBitmapK(size uint16) *bitmapK {
	asize := sizeofSub*uintptr(size)+sizeofBitmapK
	b := (*bitmapK)(unsafe.Pointer(runtime.Alloc(asize)))
	b.occupied_ = size
	return b
}
func newBitmapV(size uint16) *bitmapV {
	asize := sizeofSub*uintptr(size)+sizeofBitmapV
	b := (*bitmapV)(unsafe.Pointer(runtime.Alloc(asize)))
	b.occupied_ = size
	return b
}
func newBitmapKV(size uint16) *bitmapKV {
	asize := sizeofSub*uintptr(size)+sizeofBitmapKV
	b := (*bitmapKV)(unsafe.Pointer(runtime.Alloc(asize)))
	b.occupied_ = size
	return b
}
func makeBitmap(size int, key string, val Value, full bool) (b *bitmap_, t itrie) {
	occupied := uint16(size)
	emptystr := len(key) == 0

	switch {
	case !emptystr && full:
		Cumulative[kBitmapKV]++
		n := newBitmapKV(occupied)
		n.key_ = str(key); n.val_ = val
		b, t = &n.bitmap_, n
	case !emptystr && !full:
		Cumulative[kBitmapK]++
		n := newBitmapK(occupied)
		n.key_ = str(key)
		b, t = &n.bitmap_, n
	case emptystr && full:
		Cumulative[kBitmapV]++
		n := newBitmapV(occupied)
		n.val_ = val
		b, t = &n.bitmap_, n
	case emptystr && !full:
		Cumulative[kBitmap_]++
		n := newBitmap_(occupied)
		b, t = n, n
	}
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
	b.occupied_ = t.occupied_; b.count_ = t.count_;	b.off = t.off; b.bm = t.bm
	copy(b.sub[:b.occupied_], t.sub[:t.occupied_])
}	
func (b *bitmap_) cloneWithKey(key string) (t itrie) {
	n := newBitmapK(b.occupied_)
	Cumulative[kBitmapK]++
	n.copy(b); n.key_ = str(key)
	return n
}
func (b *bitmapV) cloneWithKey(key string) (t itrie) {
	n := newBitmapKV(b.occupied_)
	Cumulative[kBitmapKV]++
	n.copy(&b.bitmap_); n.key_ = str(key); n.val_ = b.val_
	return n
}
func (b *bitmapKV) cloneWithKey(key string) (t itrie) {
	n := newBitmapKV(b.occupied_)
	Cumulative[kBitmapKV]++
	n.copy(&b.bitmap_); n.key_ = str(key); n.val_ = b.val_
	return n
}
func (b *bitmap_) cloneWithKeyValue(key string, val Value) (t itrie, added int) {
	n := newBitmapKV(b.occupied_)
	Cumulative[kBitmapKV]++
	n.copy(b); n.key_ = str(key); n.val_ = val; n.count_++
	return n, 1
}
func (b *bitmapV) cloneWithKeyValue(key string, val Value) (t itrie, added int) {
	n := newBitmapKV(b.occupied_)
	Cumulative[kBitmapKV]++
	n.copy(&b.bitmap_); n.key_ = str(key); n.val_ = val
	return n, 0
}
func (b *bitmapKV) cloneWithKeyValue(key string, val Value) (t itrie, added int) {
	n := newBitmapKV(b.occupied_)
	Cumulative[kBitmapKV]++
	n.copy(&b.bitmap_); n.key_ = str(key); n.val_ = val
	return n, 0
}
func (b *bitmap_) modify(incr, i int, sub itrie) (t itrie) {
	n := newBitmap_(b.occupied_)
	n.copy(b); n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bitmapK) modify(incr, i int, sub itrie) (t itrie) {
	n := newBitmapK(b.occupied_)
	n.copy(&b.bitmap_); n.key_ = b.key_; n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bitmapV) modify(incr, i int, sub itrie) (t itrie) {
	n := newBitmapV(b.occupied_)
	n.copy(&b.bitmap_); n.val_ = b.val_; n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bitmapKV) modify(incr, i int, sub itrie) (t itrie) {
	n := newBitmapKV(b.occupied_)
	n.copy(&b.bitmap_); n.key_ = b.key_; n.val_ = b.val_; n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bitmap_) withoutValue() (itrie, int) {
	return b, 0
}
func (b *bitmapK) withoutValue() (itrie, int) {
	return b, 0
}
// We assume that bitmaps always have > maxBagSize children, so we don't bother checking
// if we can collapse them when removing a value.
func (b *bitmapV) withoutValue() (t itrie, removed int) {
	n := newBitmap_(b.occupied_)
	n.copy(&b.bitmap_)
	return n, 1
}
func (b *bitmapKV) withoutValue() (t itrie, removed int) {
	n := newBitmapK(b.occupied_)
	n.copy(&b.bitmap_); n.key_ = b.key_
	return n, 1
}
func (n *bitmap_) withBitmap(b *bitmap_, incr int, cb byte, r itrie) {
	n.off = b.off; n.bm = b.bm; n.count_ = b.count_ + incr
	w, bit := bitpos(uint(cb))
	exists := b.isset(w, bit)
	i := b.indexOf(w, bit)
	copy(n.sub[:i], b.sub[:i])
	src, dst := i, i
	n.sub[dst] = r; dst++; if exists { src++ } else { n.setbit(w, bit) }
	copy(n.sub[dst:n.occupied_], b.sub[src:b.occupied_])
}
func (b *bitmap_) with(incr int, cb byte, r itrie) itrie {
	t, size := b.maybeGrow(b, cb, r)
	if t != nil { return t }
	n := newBitmap_(size)
	Cumulative[kBitmap_]++
	n.withBitmap(b, incr, cb, r)
	return n
}
func (b *bitmapK) with(incr int, cb byte, r itrie) itrie {
	t, size := b.maybeGrow(b, cb, r)
	if t != nil { return t }
	n := newBitmapK(size)
	Cumulative[kBitmap_]++
	n.key_ = b.key_
	n.withBitmap(&b.bitmap_, incr, cb, r)
	return n
}
func (b *bitmapV) with(incr int, cb byte, r itrie) itrie {
	t, size := b.maybeGrow(b, cb, r)
	if t != nil { return t }
	n := newBitmapV(size)
	Cumulative[kBitmap_]++
	n.val_ = b.val_
	n.withBitmap(&b.bitmap_, incr, cb, r)
	return n
}
func (b *bitmapKV) with(incr int, cb byte, r itrie) itrie {
	t, size := b.maybeGrow(b, cb, r)
	if t != nil { return t }
	n := newBitmapKV(size)
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
func (b *bitmap_) maybeGrow(t itrie, cb byte, r itrie) (itrie, uint16) {
	// Figure out if we stay a bitmap or if we can become a span
	// we know we're too big to be a bag
	w, bit := bitpos(uint(cb))
	exists := b.isset(w, bit)
	size := b.occupied_
	if !exists {
		size++
		e := b.expanse().with(cb)
		if spanOK(e, int(size)) {
			// We can be a span
			return span(t, e, cb, r), size
		}
	}
	// still a bitmap
	return nil, size
}
func (b *bitmap_) without_(t itrie, cb byte, r itrie) itrie {
	if r == nil {
		return b.shrink(t, cb)
	}
	w, bit := bitpos(uint(cb))
	i := b.indexOf(w, bit)
	return b.modify(-1, i, r)
}
func (b *bitmap_) without(cb byte, r itrie) itrie {
	return b.without_(b, cb, r)
}
func (b *bitmapK) without(cb byte, r itrie) itrie {
	return b.without_(b, cb, r)
}
func (b *bitmapV) without(cb byte, r itrie) itrie {
	return b.without_(b, cb, r)
}
func (b *bitmapKV) without(cb byte, r itrie) itrie {
	return b.without_(b, cb, r)
}
func (b *bitmap_) shrink(t itrie, cb byte) itrie {
	// We removed a leaf -- shrink our children & possibly turn into a bag or span.
	occupied := int(b.occupied_) - 1
	e := b.expanseWithout(cb)
	if spanOK(e, occupied) {
		// We can be a span
		return spanWithout(t, e, cb)
	}
	if occupied <= maxBagSize {
		// We should become a bag
		return bagWithout(t, e, cb)
	}
	// We should stay a bitmap
	return bitmapWithout(t, e, cb)
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
func (b *bitmap_) occupied() int { return int(b.occupied_) }
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

