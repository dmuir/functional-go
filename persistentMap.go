/*
 The persistentMap package implements immutable map containers based on those found in Clojure.
*/
package persistentMap

type Value interface {
}

type Item struct {
	key string
	val Value
}

type IPersistentMap interface {
	Assoc(key string, val Value) IPersistentMap
	Without(key string) IPersistentMap
	Contains(key string) bool
	ValueAt(key string) Value
	Count() int
	Iter() chan Item
	debugPrint()
}
