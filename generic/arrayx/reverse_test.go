package arrayx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReverse(t *testing.T) {
	val := []string{"1", "2", "4", "3"}
	expected := []string{"3", "4", "2", "1"}
	assert.Equal(t, expected, Reverse(val))
}
