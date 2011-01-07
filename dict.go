/*
 The immutable package implements an immutable dictionary container inspired by
 on those found in Clojure.
*/
package immutable

type Value interface {
}

type Item struct {
	key string
	val Value
}

/*
 Dict.

 This struct implements the Dict interface via an internal itrie.
*/
type Dict struct {
	t itrie
}
func (d Dict) Assoc(key string, val Value) Dict {
	t, _ := assoc(d.t, key, val)
	return Dict{t}
}
func (d Dict) Without(key string) Dict {
	t, _ := without(d.t, key)
	return Dict{t}
}
func (d Dict) Contains(key string) bool { 
	e := entryAt(d.t, key)
	return e != nil
}
func (d Dict) ValueAt(key string) (Value, bool) {
	e := entryAt(d.t, key)
	if e != nil { return e.val(), true }
	return nil, false
}
func (d Dict) Count() int {
	if d.t != nil {
		return d.t.count()
	}
	return 0
}
func (d Dict) Foreach(fn func(string, Value)) {
	if d.t != nil {
		d.t.foreach("", fn)
	}
}
func (d Dict) Iter() chan Item {
	ch := make(chan Item)
	if d.t != nil {
		emit := func(key string, val Value) { ch <- Item{key, val} }
				
		helper := func(t itrie, emit func(string, Value)) {
			t.foreach("", emit)
			close(ch) 
		}
		go helper(d.t, emit)
	} else {
		go close(ch)
	}
	return ch
}


