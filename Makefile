include $(GOROOT)/src/Make.inc

TARG=immutable
GOFILES=\
	dict.go\
	trie.go\
	leaf.go\
	bag.go\
	span.go\
	bitmap.go\

include $(GOROOT)/src/Make.pkg
