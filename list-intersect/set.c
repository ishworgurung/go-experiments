#include <stdlib.h>

typedef struct {
  int size;
  int elems[1];
} intset;


intset *newset(int size) {
  intset *set;
  set = malloc(sizeof(intset) + sizeof(int)*(size-1));
  if (set) set->size = size;
  return set;
}


