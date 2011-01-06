package immutable

import "fmt"
import "unsafe"
import "runtime"

/*
 bag_t

 A bag is an ordered collection of sub-tries.  It may hold a single value itself.
 When a bag has 4 or more elements, it may be promoted to a span.  If it has 8 or more elements,
 it will be promoted to a span or a bitmap.
*/
type bag_ struct {
	entry_
	occupied_ uint8
	cb [maxBagSize]byte
	count_ int
	sub [maxBagSize]itrie
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

var sizeofBag_ uintptr
var sizeofBagK uintptr
var sizeofBagV uintptr
var sizeofBagKV uintptr

func init() {
	var b_ bag_
	var bk bagK
	var bv bagV
	var bkv bagKV
	var t itrie

	sizeofSub = uintptr(unsafe.Sizeof(t))
	sizeofBag_ = uintptr(unsafe.Sizeof(b_)) - maxBagSize*sizeofSub
	sizeofBagK = uintptr(unsafe.Sizeof(bk)) - maxBagSize*sizeofSub
	sizeofBagV = uintptr(unsafe.Sizeof(bv)) - maxBagSize*sizeofSub
	sizeofBagKV = uintptr(unsafe.Sizeof(bkv)) - maxBagSize*sizeofSub
}

func (b *bag_) printCBs(prefix string) {
	fmt.Printf("%s: bag.cb = [", prefix)
	for i := 0; i < int(b.occupied_); i++ {
		fmt.Printf(" %d(%c) ", b.cb[i], b.cb[i])
	}
	fmt.Printf("]\n")
}

func newBag_(size uint8) *bag_ {
	asize := sizeofSub*uintptr(size)+sizeofBag_
	b := (*bag_)(unsafe.Pointer(runtime.Alloc(asize)))
	b.occupied_ = size
	return b
}
func newBagK(size uint8) *bagK {
	asize := sizeofSub*uintptr(size)+sizeofBagK
	b := (*bagK)(unsafe.Pointer(runtime.Alloc(asize)))
	b.occupied_ = size
	return b
}
func newBagV(size uint8) *bagV {
	asize := sizeofSub*uintptr(size)+sizeofBagV
	b := (*bagV)(unsafe.Pointer(runtime.Alloc(asize)))
	b.occupied_ = size
	return b
}
func newBagKV(size uint8) *bagKV {
	asize := sizeofSub*uintptr(size)+sizeofBagKV
	b := (*bagKV)(unsafe.Pointer(runtime.Alloc(asize)))
	b.occupied_ = size
	return b
}
	
func makeBag(size uint8, key string, val Value, full bool) (*bag_, itrie) {
	emptystr := len(key) == 0

	if !emptystr {
		if full {
			Cumulative[kBagKV]++
			b := newBagKV(size)
			b.key_ = str(key); b.val_ = val; b.count_ = 1
			return &b.bag_, b
		}
		Cumulative[kBagK]++
		b := newBagK(size)
		b.key_ = str(key)
		return &b.bag_, b
	}
	if full {
		Cumulative[kBagV]++
		b := newBagV(size)
		b.val_ = val; b.count_ = 1
		return &b.bag_, b
	}
	Cumulative[kBag_]++
	b := newBag_(size)
	return b, b
}
func (b *bag_) init1(cb byte, sub itrie) {
	b.cb[0] = cb
	b.sub[0] = sub
	b.count_ += sub.count()
}	
func bag1(key string, val Value, full bool, cb byte, sub itrie) itrie {
	b, t := makeBag(uint8(1), key, val, full)
	b.init1(cb, sub)
	return t
}
func (b *bag_) init2(cb0, cb1 byte, sub0, sub1 itrie) {
	if cb1 < cb0 { cb0, cb1 = cb1, cb0; sub0, sub1 = sub1, sub0 }
	b.cb[0] = cb0; b.cb[1] = cb1 
	b.sub[0] = sub0; b.sub[1] = sub1
	b.count_ += sub0.count() + sub1.count()
}
func bag2(key string, val Value, full bool, cb0, cb1 byte, sub0, sub1 itrie) itrie {
	b, t := makeBag(uint8(2), key, val, full)
	b.init2(cb0, cb1, sub0, sub1)
	return t
}
func (b *bag_) fillWith(t itrie, cb byte, l itrie) {
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
	b, r := makeBag(uint8(t.occupied()+1), t.key(), t.val(), t.hasVal())
	b.fillWith(t, cb, l)
	return r
}
/*
 Constructs a new bag with the contents of t, minus the sub-trie at critical bit cb.  It
 is expected that any sub-trie at cb is a leaf.
*/
func (b *bag_) fillWithout(t itrie, e expanse_t, without byte) {
	index := 0
	add := func(cb byte, t itrie) { b.sub[index] = t; b.cb[index] = cb; index++ }
	t.withsubs(uint(e.low), uint(without), add)
	t.withsubs(uint(without+1), uint(e.high)+1, add)
	b.count_ = t.count() - 1
}
func bagWithout(t itrie, e expanse_t, without byte) itrie {
	size := uint8(t.occupied()-1)
	if len(t.key()) > 0 {
		if t.hasVal() {
			b := newBagKV(size)
			b.key_ = t.key(); b.val_ = t.val()
			b.fillWithout(t, e, without)
			return b
		}
		b := newBagK(size)
		b.key_ = t.key()
		b.fillWithout(t, e, without)
		return b
	}
	if t.hasVal() {
		b := newBagV(size)
		b.val_ = t.val()
		b.fillWithout(t, e, without)
		return b
	}
	b := newBag_(size)
	b.fillWithout(t, e, without)
	return b
}
func (b *bag_) copy(t *bag_) {
	b.count_ = t.count_; b.occupied_ = t.occupied_
	copy(b.cb[:t.occupied_], t.cb[:t.occupied_])
	copy(b.sub[:b.occupied_], t.sub[:t.occupied_])
}
func (b *bag_) modify(incr, i int, sub itrie) itrie {
	n := newBag_(b.occupied_)
	n.copy(b); n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bagK) modify(incr, i int, sub itrie) itrie {
	n := newBagK(b.occupied_)
	n.key_ = b.key_;
	n.copy(&b.bag_); n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bagV) modify(incr, i int, sub itrie) itrie {
	n := newBagV(b.occupied_)
	n.val_ = b.val_
	n.copy(&b.bag_); n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bagKV) modify(incr, i int, sub itrie) itrie {
	n := newBagKV(b.occupied_)
	n.key_ = b.key_; n.val_ = b.val_;
	n.copy(&b.bag_); n.count_ += incr; n.sub[i] = sub
	return n
}
func (b *bag_) cloneWithKey(key string) itrie {
	n := newBagK(b.occupied_)
	Cumulative[kBagK]++
	n.copy(b); n.key_ = str(key)
	return n
}
func (b *bagV) cloneWithKey(key string) itrie {
	n := newBagKV(b.occupied_)
	Cumulative[kBagKV]++
	n.copy(&b.bag_); n.key_ = str(key); n.val_ = b.val_
	return n
}
func (b *bagKV) cloneWithKey(key string) itrie {
	n := newBagKV(b.occupied_)
	Cumulative[kBagKV]++
	n.copy(&b.bag_); n.key_ = str(key); n.val_ = b.val_
	return n
}	
func (b *bag_) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := newBagKV(b.occupied_)
	Cumulative[kBagKV]++
	n.copy(b); n.key_ = str(key); n.val_ = val; n.count_++
	return n, 1
}
func (b *bagV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := newBagKV(b.occupied_)
	Cumulative[kBagKV]++
	n.copy(&b.bag_); n.key_ = str(key); n.val_ = val
	return n, 0
}
func (b *bagKV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := newBagKV(b.occupied_)
	Cumulative[kBagKV]++
	n.copy(&b.bag_); n.key_ = str(key); n.val_ = val
	return n, 0
}
func (b *bag_) withoutValue() (itrie, int) {
	return b, 0
}
func (b *bagK) withoutValue() (itrie, int) {
	return b, 0
}
func (b *bag_) collapse(key string) (itrie, int) {
	key += string(b.cb[0]) + b.sub[0].key()
	return b.sub[0].cloneWithKey(key), 1
}
func (b *bagV) withoutValue() (itrie, int) {
	if b.occupied_ == 1 { return b.collapse("") }
	n := newBag_(b.occupied_)
	n.copy(&b.bag_)
	return n, 1
}
func (b *bagKV) withoutValue() (itrie, int) {
	if b.occupied_ == 1 { return b.collapse(b.key_) }
	n := newBagK(b.occupied_)
	n.copy(&b.bag_); n.key_ = b.key_; n.bag_ = b.bag_
	return n, 1
}
func (n *bag_) withBag(b *bag_, incr int, size uint8, i int, cb byte, r itrie) {
	if size > maxBagSize {
		panic(fmt.Sprintf("Don't make bag's with more than %d elts.", maxBagSize))
	}
	copy(n.cb[:i], b.cb[:i])
	copy(n.sub[:i], b.sub[:i])
	src, dst := i, i
	n.cb[dst] = cb; n.sub[dst] = r; dst++
	if size == b.occupied_ { src++ }
	copy(n.cb[dst:n.occupied_], b.cb[src:b.occupied_])
	copy(n.sub[dst:n.occupied_], b.sub[src:b.occupied_])
	n.count_ = b.count_ + incr
}
func (b *bag_) with(incr int, cb byte, r itrie) itrie {
	t, size, i := b.maybeGrow(b, cb, r)
	if t != nil { return t }
	n := newBag_(size)
	Cumulative[kBag_]++
	n.withBag(b, incr, size, i, cb, r)
	return n
}
func (b *bagK) with(incr int, cb byte, r itrie) itrie {
	t, size, i := b.maybeGrow(b, cb, r)
	if t != nil { return t }
	n := newBagK(size)
	Cumulative[kBagK]++
	n.key_ = b.key_
	n.withBag(&b.bag_, incr, size, i, cb, r)
	return n
}
func (b *bagV) with(incr int, cb byte, r itrie) itrie {
	t, size, i := b.maybeGrow(b, cb, r)
	if t != nil { return t }
	n := newBagV(size)
	Cumulative[kBagV]++
	n.val_ = b.val_
	n.withBag(&b.bag_, incr, size, i, cb, r)
	return n
}
func (b *bagKV) with(incr int, cb byte, r itrie) itrie {
	t, size, i := b.maybeGrow(b, cb, r)
	if t != nil { return t }
	n := newBagKV(size)
	Cumulative[kBagKV]++
	n.key_ = b.key_; n.val_ = b.val_
	n.withBag(&b.bag_, incr, size, i, cb, r)
	return n
}
func (b *bag_) find(cb byte) (int, bool) {
	// Even though it's sorted, since len <= 7, it's almost certainly not worth it to
	// binary search.  We can still take advantage of early out.
	for i := 0; i < int(b.occupied_); i++ {
		if cb < b.cb[i] { return i, false }
		if cb == b.cb[i] { return i, true }
	}
	return int(b.occupied_), false
}
func (b *bag_) subAt(cb byte) itrie {
	i, found := b.find(cb)
	if found { return b.sub[i] }
	return nil
}
func (b *bag_) maybeGrow(t itrie, cb byte, r itrie) (itrie, uint8, int) {
	i, found := b.find(cb)
	size := b.occupied_
	if !found {
		size++
		if size >= minSpanSize {
			e := b.expanse().with(cb)
			if spanOK(e, int(size)) {
				// Prefer a span, even if we're small enough to stay a bag
				return span(t, e, cb, r), size, i
			}
		}
		if size > maxBagSize {
			// Prefer a bitmap
			return bitmap(t, cb, r), size, i
		}
	}
	return nil, size, i
}
func (b *bag_) without_(t itrie, cb byte, r itrie) itrie {
	if r == nil {
		return b.shrink(t, cb)
	}
	i, _ := b.find(cb)
	return t.modify(-1, i, r)
}
func (b *bag_) without(cb byte, r itrie) itrie {
	return b.without_(b, cb, r)
}
func (b *bagK) without(cb byte, r itrie) itrie {
	return b.without_(b, cb, r)
}
func (b *bagV) without(cb byte, r itrie) itrie {
	return b.without_(b, cb, r)
}
func (b *bagKV) without(cb byte, r itrie) itrie {
	return b.without_(b, cb, r)
}
func (b *bag_) shrink(t itrie, cb byte) itrie {
	i, _ := b.find(cb)

	// We removed a leaf -- shrink our sub-tries & possibly turn into a leaf.
	last := int(b.occupied_)-1
	if last == 0 {
		if !t.hasVal() {
			panic("we should have a value if we have no sub-tries.")
		}
		return leaf(t.key(), t.val())
	} else if last == 1 && !t.hasVal() {
		o := 1 - i
		key := t.key() + string(b.cb[o]) + b.sub[o].key()
		return b.sub[o].cloneWithKey(key)
	}
	e := b.expanse()
	if last >= minSpanSize {
		e = b.expanseWithout(cb)
		if spanOK(e, last) {
			// We can be a span
			return spanWithout(t, e, cb)
		}
	}
	// Still a bag.
	return bagWithout(t, e, cb)
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
	for i := 0; i < int(b.occupied_); i++ {
		if uint(b.cb[i]) < start { continue }
		if uint(b.cb[i]) >= end { break }
		f(b.cb[i], b.sub[i])
	}
}
func (b *bag_) count() int { return b.count_ }
func (b *bag_) occupied() int { return int(b.occupied_) }
func (b *bag_) expanse() expanse_t { return expanse(b.cb[0], b.cb[int(b.occupied_)-1]) }
func (b *bag_) expanseWithout(cb byte) expanse_t {
	if b.occupied_ == 0 { panic("Shouldn't have an empty bag.") }
	if b.occupied_ > 1 {
		last := int(b.occupied_)-1
		if cb == b.cb[0] { return expanse(b.cb[1], b.cb[last]) }
		if cb == b.cb[last] { return expanse(b.cb[0], b.cb[last-1]) }
	} else if cb == b.cb[0] { return expanse0() }
	return b.expanse()
}

