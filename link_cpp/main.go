package main

/*
#cgo CFLAGS: -I./include
#cgo LDFLAGS: -L./lib -lhello
#include <stdio.h>
#include <stdlib.h>
#include "hello.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func main() {
	name := C.CString("Jack")
	defer C.free(unsafe.Pointer(name))
	age := C.int(18)

	result := C.hello(name, age)
	fmt.Println(result)
}
