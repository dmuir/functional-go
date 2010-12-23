package persistentMap

import (
	"testing"
	"fmt"
	"reflect"
)

func TestEmptyTrie(t *testing.T) {
	m := NewTrie()
	if m.Count() != 0 {
		t.Errorf("TestEmptyMap: Count (%d) != 0", m.Count())
	}
}

func TestAssocTrie(t *testing.T) {
	m0 := NewTrie()

	m1 := m0.Assoc("foo", "bar")
	if m0.Count() != 0 {
		t.Errorf("TestAssoc: m0.Count (%d) != 0", m0.Count())
	}
	if m1.Count() != 1 {
		t.Errorf("TestAssoc: m1.Count (%d) != 1", m1.Count())
	}
		
	m2 := m1.Assoc("bar", "baz")
	if m0.Count() != 0 {
		t.Errorf("TestAssoc: m0.Count (%d) != 0", m0.Count())
	}
	if m1.Count() != 1 {
		t.Errorf("TestAssoc: m1.Count (%d) != 1", m1.Count())
	}
	if m2.Count() != 2 {
		t.Errorf("TestAssoc: m2.Count (%d) != 2", m2.Count())
	}

	if m1.ValueAt("foo").(string) != "bar" {
		t.Errorf("TestAssoc: m1.ValueAt('foo') (%s) != 'bar'",
			m1.ValueAt("foo").(string))
	}
	m1 = m1.Assoc("foo", "blah")
	if m1.ValueAt("foo").(string) != "blah" {
		t.Errorf("TestAssoc: m1.ValueAt('foo') (%s) != 'blah'",
			m1.ValueAt("foo").(string))
	}
	if m2.ValueAt("foo").(string) != "bar" {
		t.Errorf("TestAssoc: m2.ValueAt('foo') (%s) != 'bar'",
			m2.ValueAt("foo").(string))
	}
}

func printItems(m IPersistentMap) {
	typ := reflect.Typeof(m)
	fmt.Printf("Dumping map(type=%s)...\n", typ.String())
	for item := range m.Iter() {
		fmt.Printf("%s: %d\n", item.key, item.val.(int))
	}
}

func TestWithoutTrie(t *testing.T) {
	m := NewTrie()
	m = m.Assoc("A", 14)
	m = m.Assoc("K", 13)
	m = m.Assoc("Q", 12)
	m = m.Assoc("J", 11)
	m = m.Assoc("T", 10)
	m = m.Assoc("9", 9)
	m = m.Assoc("8", 8)
	m = m.Assoc("7", 7)
	m = m.Assoc("6", 6)
	m = m.Assoc("5", 5)
	m = m.Assoc("4", 4)
	m = m.Assoc("3", 3)
	m = m.Assoc("2", 2)

	if !m.Contains("T") {
		t.Errorf("TestWithout: m.Contains('T') is false.")
	}
	w := m.Without("T")
	if w.Contains("T") {
		t.Errorf("TestWithout: w.Contains('T') is true.")
	}
	if !m.Contains("T") {
		t.Errorf("TestWithout: m.Contains('T') is false.")
	}
}

func TestIterTrie(t *testing.T) {
	var keys [256]string
	m := NewTrie()

	for i := 0; i < 256; i++ {
		keys[i] = fmt.Sprintf("%02x", 255 - i)
		m = m.Assoc(keys[i], i)
	}

	if m.Count() != 256 {
		t.Errorf("TestIter: m.Count() (%d) != 256", m.Count())
	}

	count := 0
	for item := range m.Iter() {
		count++
		i := item.val.(int)
		if item.key != keys[i] {
			t.Errorf("TestIter: (%d) %s != %s", i, item.key, keys[i])
		}
	}
	if count != m.Count() {
		t.Errorf("TestIter: only iterated %d of %d items.", count, m.Count())
	}
}
		
