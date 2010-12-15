/*
 The persistentMap package implements immutable map containers similar to those found in Clojure.
*/
package persistentMap

type Value interface {
}

type IPersistentMap interface {
	Assoc(key string, val Value) IPersistentMap
	Without(key string) IPersistentMap
	//Contains(key string) bool
	//ValueAt(key string) Value
	//Empty() bool
	Count() int
}

type treeNode interface {
	key() string
	val() Value
	left() treeNode
	right() treeNode
	addLeft(treeNode) treeNode
	addRight(treeNode) treeNode
	removeLeft(treeNode) treeNode
	removeRight(treeNode) treeNode
	black() bool
	red() bool
	blacken() treeNode
	redden() treeNode
	balanceLeft(parent treeNode) treeNode
	balanceRight(parent treeNode) treeNode
	replace(key string, val Value, left treeNode, right treeNode) treeNode
}

type leafNode struct {
	_key string
	_val Value
}
func (n *leafNode) key() string { return n._key }
func (n *leafNode) val() Value { return n._val }
func (n *leafNode) left() treeNode { return nil }
func (n *leafNode) right() treeNode { return nil }
func (n *leafNode) addLeft(treeNode) treeNode { panic("Unexpected method call.") }
func (n *leafNode) addRight(treeNode) treeNode { panic("Unexpected method call.") }
func (n *leafNode) removeLeft(treeNode) treeNode { panic("Unexpected method call.") }
func (n *leafNode) removeRight(treeNode) treeNode { panic("Unexpected method call.") }
func (n *leafNode) black() bool { panic("Unexpected method call.") }
func (n *leafNode) red() bool { panic("Unexpected method call.") }
func (n *leafNode) blacken() treeNode { panic("Unexpected method call.") }
func (n *leafNode) redden() treeNode { panic("Unexpected method call.") }
func (n *leafNode) replace(string, Value, treeNode, treeNode) treeNode { panic("Unexpected method call.") }
func (n *leafNode) balanceLeft(parent treeNode) treeNode {
	return black(parent.key(), parent.val(), n, parent.right())
}
func (n *leafNode) balanceRight(parent treeNode) treeNode {
	return black(parent.key(), parent.val(), parent.left(), n)
}


type blackLeaf struct {
	leafNode
}
func newBlackLeaf(key string, val Value) *blackLeaf {
	return &blackLeaf{leafNode{key, val}}
}
func (n *blackLeaf) addLeft(ins treeNode) treeNode { return ins.balanceLeft(n) }
func (n *blackLeaf) addRight(ins treeNode) treeNode { return ins.balanceRight(n) }
func (n *blackLeaf) removeLeft(del treeNode) treeNode {
	return balanceLeftDel(n.key(), n.val(), del, n.right())
}
func (n *blackLeaf) removeRight(del treeNode) treeNode {
	return balanceRightDel(n.key(), n.val(), n.left(), del)
}
func (n *blackLeaf) black() bool { return true }
func (n *blackLeaf) red() bool { return false }
func (n *blackLeaf) blacken() treeNode { return n }
func (n *blackLeaf) redden() treeNode { return newRedLeaf(n.key(), n.val()) }
func (n *blackLeaf) replace(key string, val Value, left treeNode, right treeNode) treeNode {
	return black(key, val, left, right)
}

type blackBranch struct {
	blackLeaf
	_left treeNode
	_right treeNode
}
func newBlackBranch(key string, val Value, left treeNode, right treeNode) *blackBranch {
	return &blackBranch{blackLeaf{leafNode{key, val}}, left, right}
}
func (n *blackBranch) left() treeNode { return n._left }
func (n *blackBranch) right() treeNode { return n._right }
func (n *blackBranch) redden() treeNode { return newRedBranch(n.key(), n.val(), n.left(), n.right()) }

type redLeaf struct {
	leafNode
}
func newRedLeaf(key string, val Value) *redLeaf {
	return &redLeaf{leafNode{key, val}}
}
func (n *redLeaf) addLeft(ins treeNode) treeNode { return red(n.key(), n.val(), ins, n.right()) }
func (n *redLeaf) addRight(ins treeNode) treeNode { return red(n.key(), n.val(), n.left(), ins) }
func (n *redLeaf) removeLeft(del treeNode) treeNode { return red(n.key(), n.val(), del, n.right()) }
func (n *redLeaf) removeRight(del treeNode) treeNode { return red(n.key(), n.val(), n.left(), del) }
func (n *redLeaf) black() bool { return false }
func (n *redLeaf) red() bool { return true }
func (n *redLeaf) blacken() treeNode { return newBlackLeaf(n.key(), n.val()) }
func (n *redLeaf) redden() treeNode { panic("Invariant violation") }
func (n *redLeaf) replace(key string, val Value, left treeNode, right treeNode) treeNode {
	return red(key, val, left, right)
}

type redBranch struct {
	redLeaf
	_left treeNode
	_right treeNode
}
func newRedBranch(key string, val Value, left treeNode, right treeNode) *redBranch {
	return &redBranch{redLeaf{leafNode{key, val}}, left, right}
}
func (n *redBranch) left() treeNode { return n._left }
func (n *redBranch) right() treeNode { return n._right }
func (n *redBranch) blacken() treeNode {
	return newBlackBranch(n.key(), n.val(), n.left(), n.right())
}
func (n *redBranch) balanceLeft(parent treeNode) treeNode {
	if n.left().red() {
		return red(n.key(), n.val(),
			n.left().blacken(),
			black(parent.key(), parent.val(), n.right(), parent.right()))
	} else if n.right().red() {
		return red(n.right().key(), n.right().val(),
			black(n.key(), n.val(), n.left(), n.right().left()),
			black(parent.key(), parent.val(), n.right().right(), parent.right()))
	}
	return black(parent.key(), parent.val(), n, parent.right())
}
func (n *redBranch) balanceRight(parent treeNode) treeNode {
	if n.right().red() {
		return red(n.key(), n.val(),
			black(parent.key(), parent.val(), parent.left(), n.right()),
			n.right().blacken())
	} else if n.left().red() {
		return red(n.left().key(), n.left().val(),
			black(parent.key(), parent.val(), parent.left(), n.left().left()),
			black(n.key(), n.val(), n.left().right(), n.right()))
	}
	return black(parent.key(), parent.val(), parent.left(), n)
}

type persistentSortedMap struct {
	tree treeNode
	count int
}

func newSortedMap(tree treeNode, count int) IPersistentMap {
	m := new(persistentSortedMap)
	m.tree = tree
	m.count = count
	return m
}

func (m *persistentSortedMap) min() treeNode {
	t := m.tree
	if t != nil {
		for t.left() != nil {
			t = t.left()
		}
	}
	return t
}

func (m *persistentSortedMap) max() treeNode {
	t := m.tree
	if t != nil {
		for t.right() != nil {
			t = t.right()
		}
	}
	return t
}

func (m *persistentSortedMap) replace(t treeNode, key string, val Value) treeNode {
	if key == t.key() {
		return t.replace(t.key(), val, t.left(), t.right())
	} else if key < t.key() {
		return t.replace(t.key(), t.val(), m.replace(t.left(), key, val), t.right())
	}
	return t.replace(t.key(), t.val(), t.left(), m.replace(t.right(), key, val))
}

func (m *persistentSortedMap) Assoc(key string, val Value) IPersistentMap {
	t, found := m.add(m.tree, key, val)
	if t == nil {
		if found.val() == val {
			return m
		}
		return newSortedMap(m.replace(m.tree, key, val), m.count)
	}
	return newSortedMap(m.tree.blacken(), m.count + 1)
}

func (m *persistentSortedMap) Without(key string) IPersistentMap {
	t, found := m.remove(m.tree, key)
	if t == nil {
		if found == nil { // doesn't contain key
			return m
		}
		// empty
		return newSortedMap(nil, 0)
	}
	return newSortedMap(m.tree.blacken(), m.count - 1)
}

func (m *persistentSortedMap) Count() int {
	return m.count
}

func (m *persistentSortedMap) add(t treeNode, key string, val Value) (treeNode, treeNode) {
	if t == nil {
		return newRedLeaf(key, val), nil
	}
	if key == t.key() {
		return nil, t
	}
	if key < t.key() {
		ins, found := m.add(t.left(), key, val)
		if ins == nil {
			return nil, found
		}
		return t.addLeft(ins), found
	}
	ins, found := m.add(t.right(), key, val)
	if ins == nil {
		return nil, found
	}
	return t.addRight(ins), found
}

func (m *persistentSortedMap) remove(t treeNode, key string) (treeNode, treeNode) {
	if t == nil {
		return nil, nil
	}
	if key == t.key() {
		return append(t.left(), t.right()), t
	}
	if key < t.key() {
		del, found := m.remove(t.left(), key)
		if (del == nil) && (found == nil) {
			return nil, nil
		}
		if t.left().black() {
			return balanceLeftDel(t.key(), t.val(), del, t.right()), found
		} else {
			return red(t.key(), t.val(), del, t.right()), found
		}
	}
	del, found := m.remove(t.right(), key)
	if (del == nil) && (found == nil) {
		return nil, nil
	}
	if t.right().black() {
		return balanceRightDel(t.key(), t.val(), t.left(), del), found
	}
	return red(t.key(), t.val(), t.left(), del), found
}

func append(left treeNode, right treeNode) treeNode {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	if left.red() {
		if right.red() {
			app := append(left.right(), right.left())
			if app.red() {
				return red(app.key(), app.val(),
					red(left.key(), left.val(), left.left(), app.left()),
					red(right.key(), right.val(), app.right(), right.right()))
			}
			return red(left.key(), left.val(), left.left(),
				red(right.key(), right.val(), app, right.right()))
		}
		return red(left.key(), left.val(), left.left(), append(left.right(), right))
	} else if right.red() {
		return red(right.key(), right.val(), append(left, right.left()), right.right())
	}
	// black/black
	app := append(left.right(), right.left())
	if app.red() {
		return red(app.key(), app.val(),
			black(left.key(), left.val(), left.left(), app.left()),
			black(right.key(), right.val(), app.right(), right.right()))
	}
	return balanceLeftDel(left.key(), left.val(), left.left(),
		black(right.key(), right.val(), app, right.right()))
}

func balanceLeftDel(key string, val Value, del treeNode, right treeNode) treeNode {
	if del.red() {
		return red(key, val, del.blacken(), right)
	} else if right.black() {
		return rightBalance(key, val, del, right.redden())
	} else if right.red() && right.left().black() {
		return red(right.left().key(), right.left().val(),
			black(key, val, del, right.left().left()),
			rightBalance(right.key(), right.val(), right.left().right(), right.right().redden()))
	}
	panic("Invariant violation")
}

func balanceRightDel(key string, val Value, left treeNode, del treeNode) treeNode {
	if del.red() {
		return red(key, val, left, del.blacken())
	} else if left.black() {
		return leftBalance(key, val, left.redden(), del)
	} else if left.red() && left.right().black() {
		return red(left.right().key(), left.right().val(),
			leftBalance(left.key(), left.val(), left.left().redden(), left.right().left()),
			black(key, val, left.right().right(), del))
	}
	panic("Invariant violation")
}

func leftBalance(key string, val Value, ins treeNode, right treeNode) treeNode {
	if ins.red() && ins.left().red() {
		return red(ins.key(), ins.val(), ins.left().blacken(), black(key, val, ins.right(), right))
	} else if ins.red() && ins.right().red() {
		return red(ins.right().key(), ins.right().val(),
			black(ins.key(), ins.val(), ins.left(), ins.right().left()),
			black(key, val, ins.right().right(), right))
	}
	return black(key, val, ins, right)
}

func rightBalance(key string, val Value, left treeNode, ins treeNode) treeNode {
	if ins.red() && ins.right().red() {
		return red(ins.key(), ins.val(), black(key, val, left, ins.left()), ins.right().blacken())
	} else if ins.red() && ins.left().red() {
		return red(ins.left().key(), ins.left().val(),
			black(key, val, left, ins.left().left()),
			black(ins.key(), ins.val(), ins.left().right(), ins.right()))
	}
	return black(key, val, left, ins)
}

func red(key string, val Value, left treeNode, right treeNode) treeNode {
	if (left == nil) && (right == nil) {
		return newRedLeaf(key, val)
	}
	return newRedBranch(key, val, left, right)
}

func black(key string, val Value, left treeNode, right treeNode) treeNode {
	if (left == nil ) && (right == nil) {
		return newBlackLeaf(key, val)
	}
	return newBlackBranch(key, val, left, right)
}