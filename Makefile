include $(GOROOT)/src/Make.inc

TARG=persistentMap
GOFILES=\
	persistentMap.go\
	persistentSortedMap.go\

include $(GOROOT)/src/Make.pkg
