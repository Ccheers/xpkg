package arrayx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMerge(t *testing.T) {
	type args[T any] struct {
		slices [][]T
	}
	type testCase[T any] struct {
		name string
		args args[T]
		want []T
	}
	tests := []testCase[uint32]{
		{
			name: "1",
			args: args[uint32]{
				slices: [][]uint32{
					{1, 2, 3},
					{3, 4, 5},
				},
			},
			want: []uint32{1, 2, 3, 3, 4, 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, Merge(tt.args.slices...), "Merge(%v)", tt.args.slices)
		})
	}
}
