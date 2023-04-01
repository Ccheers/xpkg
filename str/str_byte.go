package str

import (
	"reflect"
	"unsafe"
)

func BytesToString(bts []byte) string {
	return *(*string)(unsafe.Pointer(&bts))
}

func StringToByte(s string) []byte {
	var b []byte
	l := len(s)
	p := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	p.Data = (*reflect.StringHeader)(unsafe.Pointer(&s)).Data
	p.Len = l
	p.Cap = l
	return b
}
