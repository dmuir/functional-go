include $(GOROOT)/src/Make.inc

TARG=persistentMap
GOFILES=\
	persistentMap.go\
	persistentSortedMap.go\
	persistentTrie.go\

include $(GOROOT)/src/Make.pkg
