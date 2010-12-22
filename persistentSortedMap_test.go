package persistentMap

import (
	"testing"
	"fmt"
)

func TestEmptyMap(t *testing.T) {
	m := NewSortedMap()
	if m.Count() != 0 {
		t.Error("TestEmptyMap: Count (%d) != 0", m.Count())
	}
}

func TestAssoc(t *testing.T) {
	m0 := NewSortedMap()

	m1 := m0.Assoc("foo", "bar")
	if m0.Count() != 0 {
		t.Error("TestAssoc: m0.Count (%d) != 0", m0.Count())
	}
	if m1.Count() != 1 {
		t.Error("TestAssoc: m1.Count (%d) != 1", m1.Count())
	}
		
	m2 := m1.Assoc("bar", "baz")
	if m0.Count() != 0 {
		t.Error("TestAssoc: m0.Count (%d) != 0", m0.Count())
	}
	if m1.Count() != 1 {
		t.Error("TestAssoc: m1.Count (%d) != 1", m1.Count())
	}
	if m2.Count() != 2 {
		t.Error("TestAssoc: m2.Count (%d) != 2", m2.Count())
	}

	if m1.ValueAt("foo").(string) != "bar" {
		t.Error("TestAssoc: m1.ValueAt('foo') (%s) != 'bar'",
			m1.ValueAt("foo").(string))
	}
	m1 = m1.Assoc("foo", "blah")
	if m1.ValueAt("foo").(string) != "blah" {
		t.Error("TestAssoc: m1.ValueAt('foo') (%s) != 'blah'",
			m1.ValueAt("foo").(string))
	}
	if m2.ValueAt("foo").(string) != "bar" {
		t.Error("TestAssoc: m2.ValueAt('foo') (%s) != 'bar'",
			m2.ValueAt("foo").(string))
	}
}

func TestWithout(t *testing.T) {
	m := NewSortedMap()
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
		t.Error("TestWithout: m.Contains('T') is false.")
	}
	w := m.Without("T")
	if w.Contains("T") {
		t.Error("TestWithout: w.Contains('T') is true.")
	}
	if !m.Contains("T") {
		t.Error("TestWithout: m.Contains('T') is false.")
	}
}

func TestIter(t *testing.T) {
	var keys [256]string
	m := NewSortedMap()

	for i := 0; i < 256; i++ {
		keys[i] = fmt.Sprintf("%02x", 255 - i)
		m = m.Assoc(keys[i], i)
	}

	if m.Count() != 256 {
		t.Error("TestIter: m.Count() (%d) != 256", m.Count())
	}

	printItems(m)

	count := 0
	for item := range m.Iter() {
		count++
		if item.key != keys[255 - item.val.(int)] {
			i := 255 - item.val.(int)
			t.Error("TestIter: (%d) %s != %s", i, item.key, keys[i])
		}
	}
	if count != m.Count() {
		t.Error("Didn't finish iteration.")
	}
}
		
