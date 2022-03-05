package str

import (
	"reflect"
	"unsafe"
)

func BytesToString(bts []byte) string {
	return *(*string)(unsafe.Pointer(&bts))
}

func StringToByte(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bts := &reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(bts))
}
