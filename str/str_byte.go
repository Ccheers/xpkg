package str

import (
	"unsafe"
)

func BytesToString(bts []byte) string {
	return *(*string)(unsafe.Pointer(&bts))
}

func StringToByte(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}
