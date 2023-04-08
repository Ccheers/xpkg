package arrayx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqualUnordered(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{
			"equal sets",
			[]string{"a", "b", "c"},
			[]string{"c", "b", "a"},
			true,
		},
		{
			"empty slices",
			[]string{},
			[]string{},
			true,
		},
		{
			"nil slices",
			nil,
			nil,
			true,
		},
		{
			"slices with duplicates",
			[]string{"a", "b", "a"},
			[]string{"b", "b", "b", "a"},
			false,
		},
		{
			"slices with duplicates",
			[]string{"a", "b", "a"},
			[]string{"a", "b", "b", "a"},
			false,
		},
		{
			"not equal slices",
			[]string{"a", "a", "a"},
			[]string{"b"},
			false,
		},
		{
			"a contains more elements",
			[]string{"a", "b", "c"},
			[]string{"a", "b"},
			false,
		},
		{
			"b contains more elements",
			[]string{"a", "b"},
			[]string{"a", "b", "c"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EqualUnordered(tt.a, tt.b))
		})
	}
}
