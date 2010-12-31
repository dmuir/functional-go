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

type IDict interface {
	Assoc(key string, val Value) IDict
	Without(key string) IDict
	Contains(key string) bool
	ValueAt(key string) (Value, bool)
	Count() int
	Foreach(func(string, Value))
	Iter() chan Item
}
