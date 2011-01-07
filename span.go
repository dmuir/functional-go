package immutable

import "unsafe"
import "runtime"

/*
 span_t

 A span is a trie node that simply stores an array of sub-tries, where the critical byte
 for the sub-trie is index+start.  Range's are only used for sub-tries with high density.
*/
type span_ struct {
	entry_
	start byte
	occupied_ uint16
	size uint16
	count_ int
	sub [256]itrie
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

var sizeofSpan_ uintptr
var sizeofSpanK uintptr
var sizeofSpanV uintptr
var sizeofSpanKV uintptr

func init() {
	var s_ span_
	var sk spanK
	var sv spanV
	var skv spanKV
	var t itrie

	sizeofSub = uintptr(unsafe.Sizeof(t))
	sizeofSpan_ = uintptr(unsafe.Sizeof(s_)) - 256*sizeofSub
	sizeofSpanK = uintptr(unsafe.Sizeof(sk)) - 256*sizeofSub
	sizeofSpanV = uintptr(unsafe.Sizeof(sv)) - 256*sizeofSub
	sizeofSpanKV = uintptr(unsafe.Sizeof(skv)) - 256*sizeofSub
}

func newSpan_(size uint16) *span_ {
	asize := sizeofSub*uintptr(size)+sizeofSpan_
	s := (*span_)(unsafe.Pointer(runtime.Alloc(asize)))
	s.size = size
	return s
}
func newSpanK(size uint16) *spanK {
	asize := sizeofSub*uintptr(size)+sizeofSpanK
	s := (*spanK)(unsafe.Pointer(runtime.Alloc(asize)))
	s.size = size
	return s
}
func newSpanV(size uint16) *spanV {
	asize := sizeofSub*uintptr(size)+sizeofSpanV
	s := (*spanV)(unsafe.Pointer(runtime.Alloc(asize)))
	s.size = size
	return s
}
func newSpanKV(size uint16) *spanKV {
	asize := sizeofSub*uintptr(size)+sizeofSpanKV
	s := (*spanKV)(unsafe.Pointer(runtime.Alloc(asize)))
	s.size = size
	return s
}
func makeSpan(e expanse_t, key string, val Value, full bool) (s *span_, t itrie) {
	size := e.size
	emptystr := len(key) == 0

	switch {
	case !emptystr && full:
		Cumulative[kSpanKV]++
		n := newSpanKV(size)
		n.key_ = str(key); n.val_ = val
		s, t = &n.span_, n
	case !emptystr && !full:
		Cumulative[kSpanK]++
		n := newSpanK(size)
		n.key_ = str(key)
		s, t = &n.span_, n
	case emptystr && full:
		Cumulative[kSpanV]++
		n := newSpanV(size)
		n.val_ = val
		s, t = &n.span_, n
	case emptystr && !full:
		Cumulative[kSpan_]++
		n := newSpan_(size)
		s, t = n, n
	}
	s.start = e.low
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
	s.size = t.size; s.start = t.start; s.count_ = t.count_; s.occupied_ = t.occupied_
	copy(s.sub[:s.size], t.sub[:t.size])
}
func (s *span_) cloneWithKey(key string) itrie {
	n := newSpanK(s.size)
	Cumulative[kSpanK]++
	n.copy(s); n.key_ = str(key)
	return n
}
func (s *spanV) cloneWithKey(key string) itrie {
	n := newSpanKV(s.size)
	Cumulative[kSpanKV]++
	n.copy(&s.span_); n.key_ = str(key); n.val_ = s.val_
	return n
}
func (s *spanKV) cloneWithKey(key string) itrie {
	n := newSpanKV(s.size)
	Cumulative[kSpanKV]++
	n.copy(&s.span_); n.key_ = str(key); n.val_ = s.val_
	return n
}
func (s *span_) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := newSpanKV(s.size)
	Cumulative[kSpanKV]++
	n.copy(s); n.key_ = str(key); n.val_ = val; n.count_++
	return n, 1
}
func (s *spanV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := newSpanKV(s.size)
	Cumulative[kSpanKV]++
	n.copy(&s.span_); n.key_ = str(key); n.val_ = val
	return n, 0
}
func (s *spanKV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := newSpanKV(s.size)
	Cumulative[kSpanKV]++
	n.copy(&s.span_); n.key_ = str(key); n.val_ = val
	return n, 0
}
func (s *span_) modify(incr, i int, sub itrie) itrie {
	n := newSpan_(s.size)
	n.copy(s); n.count_ += incr; n.sub[i] = sub
	return n
}
func (s *spanK) modify(incr, i int, sub itrie) itrie {
	n := newSpanK(s.size)
	n.key_ = s.key_
	n.copy(&s.span_); n.count_ += incr; n.sub[i] = sub
	return n
}
func (s *spanV) modify(incr, i int, sub itrie) itrie {
	n := newSpanV(s.size)
	n.val_ = s.val_
	n.copy(&s.span_); n.count_ += incr; n.sub[i] = sub
	return n
}
func (s *spanKV) modify(incr, i int, sub itrie) itrie {
	n := newSpanKV(s.size)
	n.key_ = s.key_; n.val_ = s.val_
	n.copy(&s.span_); n.count_ += incr; n.sub[i] = sub
	return n
}
func (s *span_) withoutValue() (itrie, int) {
	return s, 0
}
func (s *spanK) withoutValue() (itrie, int) {
	return s, 0
}
func (s *span_) collapse(key string) (itrie, int) {
	for i, t := range s.sub {
		if t != nil {
			key += string(byte(i)+s.start) + t.key()
			return t.cloneWithKey(key), 1
			break 
		}
	}
	panic("Should always find one sub-trie to collapse to.")
}
func (s *spanV) withoutValue() (itrie, int) {
	if s.occupied_ == 1 { return s.collapse("") }
	n := newSpan_(s.size)
	n.copy(&s.span_); n.count_--
	return n, 1
}
func (s *spanKV) withoutValue() (itrie, int) {
	if s.occupied_ == 1 { return s.collapse(s.key_) }
	n := newSpanK(s.size)
	n.copy(&s.span_); n.key_ = s.key_; n.count_--
	return n, 1
}
func (n *span_) withSpan(s *span_, incr int, e expanse_t, cb byte, r itrie) {
	if e.low > s.start { panic("new start must be <= old start") }
	if int(e.size) < int(s.size) { panic("new size must be >= old size") }
	n.start = e.low; n.count_ = s.count_ + incr; n.occupied_ = s.occupied_
	copy(n.sub[s.start - n.start:n.size], s.sub[:s.size])
	i := int(cb - n.start)
	o := n.sub[i]; n.sub[i] = r
	if o == nil { n.occupied_++ }
}
func (s *span_) with(incr int, cb byte, r itrie) itrie {
	t, e := s.maybeGrow(s, cb, r)
	if t != nil { return t }
	n := newSpan_(e.size)
	Cumulative[kSpan_]++
	n.withSpan(s, incr, e, cb, r)
	return n
}
func (s *spanK) with(incr int, cb byte, r itrie) itrie {
	t, e := s.maybeGrow(s, cb, r)
	if t != nil { return t }
	n := newSpanK(e.size)
	Cumulative[kSpanK]++
	n.key_ = s.key_
	n.withSpan(&s.span_, incr, e, cb, r)
	return n
}
func (s *spanV) with(incr int, cb byte, r itrie) itrie {
	t, e := s.maybeGrow(s, cb, r)
	if t != nil { return t }
	n := newSpanV(e.size)
	Cumulative[kSpanV]++
	n.val_ = s.val_
	n.withSpan(&s.span_, incr, e, cb, r)
	return n
}
func (s *spanKV) with(incr int, cb byte, r itrie) itrie {
	t, e := s.maybeGrow(s, cb, r)
	if t != nil { return t }
	n := newSpanKV(e.size)
	Cumulative[kSpanKV]++
	n.key_ = s.key_; n.val_ = s.val_
	n.withSpan(&s.span_, incr, e, cb, r)
	return n
}
func (s *span_) subAt(cb byte) itrie {
	i := int(cb) - int(s.start)
	if i < 0 || i >= int(s.size) { return nil }
	return s.sub[i]
}
func (s *span_) maybeGrow(t itrie, cb byte, r itrie) (itrie, expanse_t) {
	// Update expanse
	e0 := s.expanse()
	e := e0.with(cb)
	
	if e.size > e0.size {
		// Figure out if we're a span, a bag, or a bitmap.
		count := int(s.occupied_)+1
		if !spanOK(e, count) {
			// We're not a span.
			if count <= maxBagSize {
				return bag(t, cb, r), e
			}
			// Prefer a bitmap
			return bitmap(t, cb, r), e
		}
	}
	
	return nil, e
}
func (s *span_) firstAfter(i int) byte {
	i++
	for ; i < int(s.size); i++ {
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
		d := s.lastBefore(int(s.size)-1)
		e.high = e.low + byte(d)
		e.size = uint16(d+1)
	}
	return e
}
func (s *span_) without_(t itrie, cb byte, r itrie) itrie {
	if r == nil {
		return s.shrink(t, cb)
	}
	i := int(cb) - int(s.start)
	return s.modify(-1, i, r)
}
func (s *span_) without(cb byte, r itrie) itrie {
	return s.without_(s, cb, r)
}
func (s *spanK) without(cb byte, r itrie) itrie {
	return s.without_(s, cb, r)
}
func (s *spanV) without(cb byte, r itrie) itrie {
	return s.without_(s, cb, r)
}
func (s *spanKV) without(cb byte, r itrie) itrie {
	return s.without_(s, cb, r)
}
func (s *span_) shrink(t itrie, cb byte) itrie {
	// We removed a leaf -- shrink our children & possibly turn into a bag or leaf.
	occupied := s.occupied_ - 1
	i := int(cb) - int(s.start)
	// We shouldn't actually let spans get small enough to hit either of the next
	// two cases
	if occupied == 0 {
		if !t.hasVal() { panic("we should have a value if we have no sub-tries.") }
		return leaf(t.key(), t.val())
	} 
	if occupied == 1 && !t.hasVal() {
		o := 0
		for ; o < int(s.size); o++ {
			if o != i && s.sub[o] != nil { break }
		}
		if o >= int(s.size) { panic("We should have another valid sub-trie") }
		key := t.key() + string(cb) + s.sub[o].key()
		return s.sub[o].cloneWithKey(key)
	}
	e := s.expanse()
	if occupied >= minSpanSize {
		e = s.expanseWithout(cb)
		if spanOK(e, int(occupied)) {
			// We can stay a span
			return spanWithout(t, e, cb)
		}
	}
	if occupied <= maxBagSize {
		// We should become a bag
		return bagWithout(t, e, cb)
	}
	// Looks like we're a bitmap
	return bitmapWithout(t, e, cb)
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
	start = uint(min(max(0, int(start) - int(s.start)), int(s.size)))
	end = uint(min(max(0, int(end) - int(s.start)), int(s.size)))
	if start >= end { return }
	for i, t := range s.sub[start:end] {
		if t == nil { continue }
		cb := s.start + byte(start) + byte(i)
		f(cb, t)
	}
}
func (s *span_) count() int { return s.count_ }
func (s *span_) occupied() int { return int(s.occupied_) }
func (s *span_) expanse() expanse_t { return expanse(s.start, s.start+byte(s.size-1)) }

