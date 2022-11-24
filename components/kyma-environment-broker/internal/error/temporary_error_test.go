package error

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemporaryError(t *testing.T) {
	// given
	err1 := fmt.Errorf("some error: %s", "argErr")
	err2 := fmt.Errorf("some error: %s", "argErr")
	err3 := NewTemporaryError("some error: %s", fmt.Errorf("argErr"))

	// when
	e1 := fmt.Errorf("wrap err %s: %w", "arg1", err1)
	e2 := AsTemporaryError(err2, "wrap err %s", "arg1")
	e3 := fmt.Errorf("wrap err %s: %w", "arg1", err3)

	// then
	assert.False(t, IsTemporaryError(e1))
	assert.True(t, IsTemporaryError(e2))
	assert.True(t, IsTemporaryError(e3))

	assert.Equal(t, "wrap err arg1: some error: argErr", e1.Error())
	assert.Equal(t, "wrap err arg1: some error: argErr", e2.Error())
	assert.Equal(t, "wrap err arg1: some error: argErr", e3.Error())
}
