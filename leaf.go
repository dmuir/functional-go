package immutable

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
		l.key_ = str(key); l.val_ = val
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
	return nil, 1
}
func (l *leafV) subAt(cb byte) itrie { return nil }
func (l *leafV) with(incr int, cb byte, r itrie) itrie {
	return bag1("", l.val_, true, cb, r)
}
func (l *leafKV) with(incr int, cb byte, r itrie) itrie {
	return bag1(l.key_, l.val_, true, cb, r)
}
func (l *leafV) without(cb byte, r itrie) itrie {
	panic("leaves can't do 'without'.")
}
func (l *leafV) foreach(prefix string, f func(string, Value)) {
	f(prefix, l.val_)
}
func (l *leafKV) foreach(prefix string, f func(string, Value)) {
	f(prefix + l.key_, l.val_)
}
func (l *leafV) withsubs(start uint, end uint, fn func(byte, itrie)) {}
func (l *leafV) key() string { return "" }
func (l *leafKV) key() string { return l.key_ }
func (l *leafV) count() int { return 1 }
func (l *leafV) occupied() int { return 0 }
func (l *leafV) expanse() expanse_t { return expanse0() }
func (l *leafV) expanseWithout(byte) expanse_t { return expanse0() }
