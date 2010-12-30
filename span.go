package immutable

import "fmt"

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
			n.key_ = str(key); n.val_ = val; n.count_ = 1
			s, t = &n.span_, n
		} else {
			Cumulative[kSpanK]++
			n := new(spanK)
			n.key_ = str(key)
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
	Cumulative[kSpanK]++
	n.span_ = *s; n.key_ = str(key)
	return n
}
func (s *spanV) cloneWithKey(key string) itrie {
	n := new(spanKV)
	Cumulative[kSpanKV]++
	n.span_ = s.span_; n.key_ = str(key); n.val_ = s.val_
	return n
}
func (s *spanKV) cloneWithKey(key string) itrie {
	n := new(spanKV)
	Cumulative[kSpanKV]++
	n.span_ = s.span_; n.key_ = str(key); n.val_ = s.val_
	return n
}
func (s *span_) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := new(spanKV)
	Cumulative[kSpanKV]++
	n.span_ = *s; n.key_ = str(key); n.val_ = val
	return n, 1
}
func (s *spanV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := new(spanKV)
	Cumulative[kSpanKV]++
	n.span_ = s.span_; n.key_ = str(key); n.val_ = val
	return n, 0
}
func (s *spanKV) cloneWithKeyValue(key string, val Value) (itrie, int) {
	n := new(spanKV)
	Cumulative[kSpanKV]++
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
	Cumulative[kSpan_]++
	added := n.with_(s, e, cb, key, val)
	return n, added
}
func (s *spanK) with(e expanse_t, cb byte, key string, val Value) (itrie, int) {
	n := new(spanK)
	Cumulative[kSpanK]++
	n.key_ = s.key_
	added := n.with_(&s.span_, e, cb, key, val)
	return n, added
}
func (s *spanV) with(e expanse_t, cb byte, key string, val Value) (itrie, int) {
	n := new(spanV)
	Cumulative[kSpanV]++
	n.val_ = s.val_
	added := n.with_(&s.span_, e, cb, key, val)
	return n, added
}
func (s *spanKV) with(e expanse_t, cb byte, key string, val Value) (itrie, int) {
	n := new(spanKV)
	Cumulative[kSpanKV]++
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

