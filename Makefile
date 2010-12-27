include $(GOROOT)/src/Make.inc

TARG=immutable
GOFILES=\
	dict.go\
	trie.go\

include $(GOROOT)/src/Make.pkg
