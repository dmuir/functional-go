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
		l.key_ = key; l.val_ = val
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
	panic("Can't have a leaf without a value")
}
func (l *leafV) assoc(t itrie, prefix string, cb byte, rest string, val Value) (itrie, int) {
	return bag1(prefix, l.val(), true, cb, leaf(rest, val)), 1
}
func (l *leafV) without(t itrie, key string) (itrie, int) {
	if len(key) == 0 { return nil, 1 }
	return t, 0
}
func (l *leafKV) without(t itrie, key string) (itrie, int) {
	if key == l.key_ { return nil, 1 }
	return t, 0
}
func (l *leafV) entryAt(key string) itrie {
	if len(key) == 0 { return l }
	return nil
}
func (l *leafKV) entryAt(key string) itrie {
	if key == l.key_ { return l }
	return nil
}
func (l *leafV) foreach(prefix string, f func(string, Value)) {
	f(prefix, l.val_)
}
func (l *leafKV) foreach(prefix string, f func(string, Value)) {
	f(prefix + l.key_, l.val_)
}
func (l *leafV) withsubs(start uint, end uint, fn func(byte, itrie)) {}
func (l *leafKV) key() string { return l.key_ }
func (l *leafV) count() int { return 1 }
func (l *leafV) occupied() int { return 0 }
func (l *leafV) expanse() expanse_t { return expanse0() }
func (l *leafV) expanseWithout(byte) expanse_t { return expanse0() }
