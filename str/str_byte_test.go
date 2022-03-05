package str

import (
	"reflect"
	"testing"
)

//
//
//
//        ***************************     ***************************         *********      ************************
//      *****************************    ******************************      *********      *************************
//     *****************************     *******************************     *********     *************************
//    *********                         *********                *******    *********     *********
//    ********                          *********               ********    *********     ********
//   ********     ******************   *********  *********************    *********     *********
//   ********     *****************    *********  ********************     *********     ********
//  ********      ****************    *********     ****************      *********     *********
//  ********                          *********      ********             *********     ********
// *********                         *********         ******            *********     *********
// ******************************    *********          *******          *********     *************************
//  ****************************    *********            *******        *********      *************************
//    **************************    *********              ******       *********         *********************
//
//

func TestBytesToString(t *testing.T) {
	type args struct {
		bts []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "1",
			args: args{
				bts: []byte("hello"),
			},
			want: "hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BytesToString(tt.args.bts); got != tt.want {
				t.Errorf("BytesToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringToByte(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "1",
			args: args{
				s: "hello",
			},
			want: []byte("hello"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringToByte(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringToByte() = %v, want %v", got, tt.want)
			}
		})
	}
}
