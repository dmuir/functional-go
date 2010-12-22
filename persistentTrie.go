package persistentMap

import (
	"fmt"
)

const maxLinear = 8
const minRangeSize = 4
const maxRangeWaste = 4

func str(s string) string {
	// We do this to ensure that the string is a new copy and not a slice of a larger string
	// We don't just return a byte slice since a byte slice is larger than a string.
	// I really wish there was a better way to do this, since we're already creating a lot
	// of work for the GC.
	bytes := []byte(s)
	return string(bytes)
}

func abs(x int) int {
	if x < 0 { return -x }
	return x
}

func min(a, b int) int {
	if a < b { return a }
	return b
}

func max(a, b int) int {
	if a > b { return a }
	return b
}

func expanse(first byte, rest ... byte) (byte, byte) {
	low := first
	high := first
	for _, v := range rest {
		if v < low { low = v }
		if v > high { high = v }
	}
	return high - low + 1, low
}

/*
 finds the location of the critical byte for 2 strings -- this is the first byte at which the
 strings differ.  The second return value indicates whether the strings match exactly.
*/
func findcrit(a string, b string) (int, bool) {
	la := len(a)
	lb := len(b)
	l := min(la, lb)
	i := 0
	for ; i < l; i++ {
		if a[i] != b[i] {
			return i, false
		}
	}
	return i, la == lb
}

/*
 split a string at the critical byte, returning the portion of the string preceeding the critical
 byte, the critical byte, and the portion after the critical byte.
*/
func splitKey(key string, crit int) (string, byte, string) {
	if crit >= len(key) {
		return key, 0, ""
	}
	return key[0:crit], key[crit], key[crit+1:]
}

type trie struct {}
func (t *trie) Assoc(string, Value) IPersistentMap { panic("Abstract Base") }
func (t *trie) Without(string) IPersistentMap { panic("Abstract Base") }
func (t *trie) Contains(string) bool { panic("Abstract Base") }
func (t *trie) ValueAt(string) Value { panic("Abstract Base") }
func (t *trie) Count() int { panic("Abstract Base") }
func (t *trie) inorder(string, chan Item) { panic("Abstract Base") }

type emptyNode struct {
	trie
}
var empty *emptyNode = &emptyNode{trie{}}
func makeChildren(size int) []IPersistentMap {
	c := make([]IPersistentMap, size)
	for i := 0; i < len(c); i++ {
		c[i] = empty
	}
	return c
}

type leaf struct {
	trie
	key string
	val Value
}
func makeLeaf(key string, val Value) IPersistentMap {
	return &leaf{trie{}, str(key), val}
}

type linearNode struct {
	trie
	key string
	val Value
	full bool
	crit string
	children []IPersistentMap
}
func critbytes(first byte, rest ... byte) string {
	return string(first) + string(rest)
}
func (l *linearNode) grow(size int, i int, ch byte) *linearNode {
	n := new(linearNode)
	*n = *l
	n.children = make([]IPersistentMap, size)
	crit := make([]byte, size)
	copy(n.children, l.children)
	copy(crit, l.crit)
	crit[i] = ch
	n.crit = string(crit)
	for i := len(l.children); i < size; i++ {
		n.children[i] = empty
	}
	return n
}
	
func addchild(first IPersistentMap, rest ... IPersistentMap) []IPersistentMap {
	children := make([]IPersistentMap, len(rest)+1)
	children[0] = first
	copy(children[1:], rest)
	return children
}
func linear(key string, val Value, full bool,
	crit string, children ... IPersistentMap) IPersistentMap {
	return &linearNode{trie{}, str(key), val, full, crit, children}
}

type rangeNode struct {
	trie
	key string
	val Value
	full bool
	start byte
	occupied byte
	children []IPersistentMap
}
func makeRange(key string, val Value, full bool, start byte, occupied byte,
	children []IPersistentMap) IPersistentMap {
	count := 0
	for _, child := range children {
		if child != nil { count++ }
	}
	if count > int(occupied) {
		fmt.Printf("count: %d, occupied: %d\n", count, occupied)
		panic("Count > Occupied")
	}
	return &rangeNode{trie{}, str(key), val, full, start, occupied, children}
}
func fillSpan(start byte, size byte, crits string, children []IPersistentMap) []IPersistentMap {
	span := makeChildren(int(size))
	for i := 0; i < len(children); i++ {
		span[crits[i] - start] = children[i]
	}
	return span
}

type bitmapNode struct {
	trie
	key string
	val Value
	full bool
	start byte
	end byte
	off [4]uint8
	bm [4]uint64
	children []IPersistentMap
}
/*
 population count implementation taken from http://www.wikipedia.org/wiki/Hamming_weight
*/
const m1  = 0x5555555555555555
const m2  = 0x3333333333333333
const m4  = 0x0f0f0f0f0f0f0f0f
const h01 = 0x0101010101010101
func countbits(bits uint64) byte {
	bits -= (bits >> 1) & m1
	bits = (bits & m2) + ((bits >> 2) & m2)
	bits = (bits + (bits >> 4)) & m4
	return byte((bits*h01)>>56)
}
func (b *bitmapNode) grow(num int) *bitmapNode {
	n := new(bitmapNode)
	*n = *b
	n.children = make([]IPersistentMap, num)
	copy(n.children, b.children)
	return n
}

func bitmapFromLinear(n *linearNode, ch byte, key string, val Value) IPersistentMap {
	// We know that the key/value pair doesn't belong to one of the existing children.
	b := new(bitmapNode)
	b.key = n.key
	b.val = n.val
	b.full = n.full
	b.children = make([]IPersistentMap, len(n.children)+1)

	// Loop through once to set up our bitmaps.  We need to do this to set up
	// the offsets & make sure the bitmaps are complete before we start using them.
	// Since linear nodes are unsorted, I don't think there's a way to short circuit this.
	for i := 0; i < len(n.children); i++ {
		crit := uint(n.crit[i])
		w := crit >> 6
		bit := uint64(1) << (crit & 0x3f)
		b.bm[w] |= bit
		b.off[w] += 1
	}
	// Update the bitmap for the new child branch
	crit := uint(ch)
	start := crit
	end := crit
	w := crit >> 6
	bit := uint64(1) << (crit & 0x3f)
	b.bm[w] |= bit
	b.off[w] += 1
	// Shift the offsets -- we don't actually care about the count in bm[3], and the 
	// offset for bm[0] should be 0.
	b.off[3] = b.off[2] + b.off[1] + b.off[0]
	b.off[2] = b.off[1] + b.off[0]
	b.off[1] = b.off[0]
	b.off[0] = 0
	// Add the child branches.
	// First the new leaf that made us expand the previous node.
	mask := bit - 1
	index := countbits(b.bm[w] & mask) + b.off[w]
	b.children[index] = makeLeaf(key, val)
	// Now the add the previous node's children.
	for i := 0; i < len(n.children); i++ {
		crit := uint(n.crit[i])
		if crit < start { start = crit }
		if crit > end { end = crit }
		w := crit >> 6
		bit := uint64(1) << (crit & 0x3f)
		mask := bit - 1
		index := countbits(b.bm[w] & mask) + b.off[w]
		b.children[index] = n.children[i]
	}
	b.start = byte(start)
	b.end = byte(end)
	return b
}
func bitmapFromRange(n *rangeNode, ch byte, key string, val Value) IPersistentMap {
	// We know that the key/value pair doesn't belong to one of the existing children.
	b := new(bitmapNode)
	b.key = n.key
	b.val = n.val
	b.full = n.full
	b.children = make([]IPersistentMap, len(n.children)+1)

	// Update the bitmaps for the new child branch.  We know that the child branch
	// falls outside of the original range, since otherwise we would not be turning
	// the range into a bitmap.
	crit := uint(ch)
	start := crit
	end := crit
	w := crit >> 6
	bit := uint64(1) << (crit & 0x3f)
	b.bm[w] |= bit
	b.off[w] += 1
	index := 0
	if ch < n.start {
		// New branch comes before the range -- it gets index 0
		b.children[index] = makeLeaf(key, val)
		index++
	}
	// Now handle the range's children.
	for i := 0; i < len(n.children); i++ {
		crit := uint(i+int(n.start))
		if crit < start { start = crit }
		if crit > end { end = crit }
		w := crit >> 6
		bit := uint64(1) << (crit & 0x3f)
		b.bm[w] |= bit
		b.off[w] += 1
		b.children[index] = n.children[i]
		index++
	}
	// Shift the offsets -- we don't actually care about the count in bm[3], and the 
	// offset for bm[0] should be 0.
	b.off[3] = b.off[2] + b.off[1] + b.off[0]
	b.off[2] = b.off[1] + b.off[0]
	b.off[1] = b.off[0]
	b.off[0] = 0
	if ch >= (n.start + byte(len(n.children))) {
		// New branch comes after the range -- it gets the last index
		b.children[index] = makeLeaf(key, val)
	}
	b.start = byte(start)
	b.end = byte(end)
	return b
}

func assoc(m IPersistentMap, key string, val Value) (IPersistentMap, byte) {
	if m != nil {
		return m.Assoc(key, val), 0
	}
	return makeLeaf(key, val), 1
}

func (e *emptyNode) Assoc(key string, val Value) IPersistentMap {
	return makeLeaf(key, val)
}

func (l *leaf) Assoc(key string, val Value) IPersistentMap {
	crit, match := findcrit(key, l.key)
	if match {
		return makeLeaf(key, val)
	}

	prefix, ch, rest := splitKey(key, crit)
	_, _ch, _rest := splitKey(l.key, crit)
	if crit == len(key) {
		return linear(prefix, val, true, critbytes(_ch),
			makeLeaf(_rest, l.val))
	} else if crit == len(l.key) {
		return linear(prefix, l.val, true, critbytes(ch),
			makeLeaf(rest, val))
	}
	return linear(prefix, nil, false, critbytes(ch, _ch),
		makeLeaf(rest, val), makeLeaf(_rest, l.val))
}

func linearFind(ch byte, crit string, children []IPersistentMap) int {
	for i := 0; i < len(children); i++ {
		if ch == crit[i] {
			return i
		}
	}
	return len(children)
}

func (n *linearNode) Assoc(key string, val Value) IPersistentMap {
	crit, match := findcrit(key, n.key)
	if match {
		return linear(key, val, true, n.crit, n.children...)
	}
	
	prefix, ch, rest := splitKey(key, crit)
	_, _ch, _rest := splitKey(n.key, crit)
	if crit < len(n.key) {
		return linear(prefix, nil, false, critbytes(ch, _ch), makeLeaf(rest, val),
			linear(_rest, n.val, n.full, n.crit, n.children...))
	}
	i := linearFind(ch, n.crit, n.children)
	if i >= len(n.children) {
		// Determine whether we're a linear, range, or a bitmap
		if i > minRangeSize {
			span, start := expanse(ch, []byte(n.crit)...)
			if int(span) <= (i+maxRangeWaste) {
				// Prefer a range, even if we're small enough to stay linear
				children := fillSpan(start, span, n.crit, n.children)
				children[ch-start] = makeLeaf(rest, val)
				return makeRange(n.key, n.val, n.full, start, byte(i+1), children)
			}
		} else if i > maxLinear {
			// Prefer a bitmap
			return bitmapFromLinear(n, ch, rest, val)
		}
	}
		
	// We still fit in a linear node
	r := n.grow(max(i+1, len(n.children)), i, ch)
	r.children[i], _ = assoc(r.children[i], rest, val)
	return r
}

func (n *rangeNode) Assoc(key string, val Value) IPersistentMap {
	crit, match := findcrit(key, n.key)
	if match {
		return makeRange(key, val, true, n.start, n.occupied, n.children)
	}

	prefix, ch, rest := splitKey(key, crit)
	_, _ch, _rest := splitKey(n.key, crit)
	if crit < len(n.key) {
		return linear(prefix, nil, false, critbytes(ch, _ch), makeLeaf(rest, val),
			makeRange(_rest, n.val, n.full, n.start, n.occupied, n.children))
	}
	// Update expanse
	oldspan := byte(len(n.children))
	span, start := expanse(ch, n.start, n.start + oldspan-1)
	
	if span > oldspan {
		// Figure out if we're a range, a linear, or a bitmap.
		count := n.occupied+1
		if span > (count+maxRangeWaste) {
			// We're not a range.
			if count <= maxLinear {
				// Prefer a linear
				crits := make([]byte, count)
				crits[0] = ch
				children := make([]IPersistentMap, count)
				children[0] = makeLeaf(rest, val)
				next := 1
				for i, child := range n.children {
					if child != nil && child != empty {
						crits[next] = n.start + byte(i)
						children[next] = child
						next++
					}
				}
				return linear(n.key, n.val, n.full, string(crits), children...)
			}
			// Prefer a bitmap
			return bitmapFromRange(n, ch, rest, val)
		}
	}

	// Prefer a range -- the code below handles the case of adding a new child, or
	// overwriting an existing one.
	if start > n.start { panic("new start must be <= old start") }
	if span < oldspan { panic("new span must be >= old span") }
	children := makeChildren(int(span))
	copy(children[n.start - start:], n.children)
	child, added := assoc(children[ch-start], rest, val)
	children[ch - start] = child
	return makeRange(n.key, n.val, n.full, start, n.occupied+added, children)
}

func (n *bitmapNode) Assoc(key string, val Value) IPersistentMap {
	crit, match := findcrit(key, n.key)
	if match {
		b := n.grow(len(n.children))
		b.key = key
		b.val = val
		b.full = true
		return b
	}

	prefix, ch, rest := splitKey(key, crit)
	_, _ch, _rest := splitKey(n.key, crit)
	if crit < len(n.key) {
		b := n.grow(len(n.children))
		b.key = _rest
		return linear(prefix, nil, false, critbytes(ch, _ch), makeLeaf(rest, val), b)
	}

	// Figure out if we stay a bitmap or if we can become a range
	// we know we're too big to become linear
	span, start := expanse(n.start, n.end, ch)
	w := ch >> 6
	bit := uint64(1) << (ch & 0x3f)
	count := len(n.children)
	replace := n.bm[w] & bit != 0
	if !replace { count++ }
	if replace || int(span) > (count+maxRangeWaste) {
		// We stay a bitmap
		b := n.grow(count)
		if !replace {
			// New -- update bitmap & offsets
			b.bm[w] |= bit
			for ; w < 3; w++ {
				b.off[w+1]++
			}
		}
		index := countbits(n.bm[w] & (bit-1)) + n.off[w]
		b.children[index], _ = assoc(n.children[index], rest, val)
		return b
	}

	// We can be a range
	children := makeChildren(int(span))
	index := 0
	for w:=0; w < 4; w++ {
		bm := n.bm[w]
		for bm != 0 {
			bit := bm ^ (bm - 1)
			dst := (countbits(bit-1) + byte(64*w)) - start
			bm &= (bm-1)
			children[dst] = n.children[index]
			index++
		}
	}
	children[ch-start] = makeLeaf(rest, val)
	return makeRange(n.key, n.val, n.full, start, byte(index+1), children)
}

func (e *emptyNode) Without(key string) IPersistentMap {
	return e
}

func (l *leaf) Without(key string) IPersistentMap {
	if key == l.key { return empty }
	return l
}

func (n *linearNode) Without(key string) IPersistentMap {
	crit, match := findcrit(key, n.key)
	if crit < len(n.key) {
		// we don't have the element being removed
		return n
	}

	if match {
		return linear(key, nil, false, n.crit, n.children...)
	}

	_, ch, rest := splitKey(key, crit)
	i := linearFind(ch, n.crit, n.children)
	if i >= len(n.children) {
		// we don't have the element being removed
		return n
	}
	b := n.children[i].Without(rest)
	if b == empty {
		// We removed a leaf node -- shrink our children & possibly turn into a leaf or range
		last := len(n.children)-1
		children := make([]IPersistentMap, last)
		copy(children, n.children)
		crit := make([]byte, last)
		copy(crit, n.crit)
		children[i] = n.children[last]
		crit[i] = n.crit[last]
		if last > minRangeSize {
			span, start := expanse(crit[0], crit...)
			if int(span) < (last+maxRangeWaste) {
				rchildren := fillSpan(start, span, string(crit), children)
				return makeRange(n.key, n.val, n.full, start, byte(last), rchildren)
			}
		}
		return linear(n.key, n.val, n.full, string(crit), children...)
	}
	children := make([]IPersistentMap, len(n.children))
	copy(children, n.children)
	children[i] = b
	return linear(n.key, n.val, n.full, n.crit, children...)
}

func (e *emptyNode) Contains(key string) bool {
	return false
}

func (l *leaf) Contains(key string) bool {
	if key == l.key { return true }
	return false
}

func (n *linearNode) Contains(key string) bool {
	crit, match := findcrit(key, n.key)
	if match && n.full { return true }
	if crit >= len(key) { return false }
	_, ch, rest := splitKey(key, crit)
	i := linearFind(ch, n.crit, n.children)
	if i >= len(n.children) { return false }
	return n.children[i].Contains(rest)
}

func (n *rangeNode) Contains(key string) bool {
	crit, match := findcrit(key, n.key)
	if match && n.full { return true }
	if crit >= len(key) { return false }
	_, ch, rest := splitKey(key, crit)
	if ch >= n.start && int(ch) < (int(n.start)+len(n.children)) {
		i := ch + n.start
		if n.children[i] != nil { return n.children[i].Contains(rest) }
	}
	return false
}

func (e *emptyNode) ValueAt(key string) Value {
	return nil
}

func (l *leaf) ValueAt(key string) Value {
	if key == l.key {
		return l.val
	}
	return nil
}

func (n *linearNode) ValueAt(key string) Value {
	crit, match := findcrit(key, n.key)
	if match && n.full { return n.val }
	if crit >= len(key) { return nil }
	_, ch, rest := splitKey(key, crit)
	i := linearFind(ch, n.crit, n.children) 
	if i > len(n.children) { return nil }
	return n.children[i].ValueAt(rest)
}

func (e *emptyNode) Count() int {
	return 0
}

func (e *leaf) Count() int {
	return 1
}

func (l *linearNode) Count() int {
	count := 0
	if l.full { count++ }
	for _, c := range l.children {
		count += c.Count()
	}
	return count
}

func (e *emptyNode) inorder(prefix string, ch chan Item) {
	close(ch)
}

func (t *trie) Iter() chan Item {
	ch := make(chan Item)
	go t.inorder("", ch)
	return ch
}

func NewTrie() IPersistentMap {
	return empty
}


