package main

import "unsafe"

// #include <stdio.h>
// void myFunc(void *c) {
//   printf("Hello %s!\n", (char*) c);
// }
// void myCharPtrFunc(char *c) {
//   printf("Hello %s!\n", c);
// }
import "C"

type BufferStruct struct {
	buff []byte
}

func main() {
	msg := []byte("world!")
	C.myFunc(unsafe.Pointer(&msg[0]))

	myBuff := BufferStruct{
		buff: []byte("world2!"),
	}
	C.myCharPtrFunc((*C.char)(unsafe.Pointer(&myBuff.buff[0])))
	C.myFunc(unsafe.Pointer(&myBuff.buff[0]))
}
