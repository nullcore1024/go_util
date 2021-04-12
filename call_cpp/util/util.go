package util

/*
#cgo CFLAGS: -I./
#cgo LDFLAGS: -L./ -lssl -lcrypto
#include "util.h"
#include <stdio.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	//	"unsafe"
)

func GoSum(a, b int) int {
	s := C.sum(C.int(a), C.int(b))
	fmt.Println(s)
	return int(s)
}

func GoGreet(name string, year int) []byte {
	Name := C.CString(name)
	//defer C.free(unsafe.Pointer(name))

	Year := C.int(year)

	ptr := C.malloc(C.sizeof_char * 1024)
	//defer C.free(unsafe.Pointer(ptr))

	size := C.greet(Name, Year, (*C.char)(ptr))
	data := C.GoBytes(ptr, C.int(size))
	return data
}

func RSAEncrypt(src []byte, N, E string) []byte {
	dst := C.malloc(C.sizeof_char * 1024)
	//defer C.free(unsafe.Pointer(dst))

	var size C.int

	n := C.CString(N)
	//defer C.free(unsafe.Pointer(n))

	e := C.CString(E)
	//defer C.free(unsafe.Pointer(e))

	ret := C.rsaEncrypt((*C.char)(dst),
		(*C.int)((&size)),
		(*C.char)(C.CBytes(src)),
		C.int(len(src)),
		n, e)

	fmt.Println("ret:", ret, " size:", size)

	data := C.GoBytes(dst, C.int(size))
	return data
}
