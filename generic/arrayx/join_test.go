package arrayx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoin(t *testing.T) {
	testCases := []struct {
		input    interface{}
		glue     string
		expected string
	}{
		{[]string{"hello", "world"}, "|", "hello|world"},
		{[]int{1, 2, 3, 4, 5}, ";", "1;2;3;4;5"},
		{[]float32{3.14, 42}, "/", "3.14/42"},
		{[]string{"", "", "", "", "", "", "Batman"}, "NaN", "NaNNaNNaNNaNNaNNaNBatman"},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tc.expected, Join(tc.input, tc.glue))
		})
	}
}
