// +build cgo

package main

/*
#include <stdlib.h>
#include <stdio.h>
#include <glib.h>
#cgo LDFLAGS: -I/usr/include/glib-2.0 -I/usr/lib/glib-2.0/include -lglib-2.0 -L. -lc

typedef struct {
  int size;
  int elems[1];
} intset;

intset* newset(int size) {
	intset *set;
  	set = malloc(sizeof(intset) + sizeof(int)*(size-1));
  	if (set) {
		set->size = size;
	}
	return set;
}

intset intersect(intset* x, intset* y) {
	for(int i = 0; i < x->size; i++) {
		for(int j = 0; j < y->size; j++) {

		}
	}
	printf("%x %x \n", x, y);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func main() {
	s1 := C.newset(C.int(10))
	s2 := C.newset(C.int(5))
	C.intersect(s1, s2)
	C.free(unsafe.Pointer(s1))
	C.free(unsafe.Pointer(s2))
	fmt.Printf("%v %v", s1, s2)
}
