package error

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestTemporaryError(t *testing.T) {
	// given
	err1 := fmt.Errorf("some error: %s", "argErr")
	err2 := fmt.Errorf("some error: %s", "argErr")
	err3 := NewTemporaryError("some error: %s", fmt.Errorf("argErr"))

	// when
	e1 := errors.Wrapf(err1, "wrap err %s", "arg1")
	e2 := AsTemporaryError(err2, "wrap err %s", "arg1")
	e3 := errors.Wrapf(err3, "wrap err %s", "arg1")

	// then
	assert.False(t, IsTemporaryError(e1))
	assert.True(t, IsTemporaryError(e2))
	assert.True(t, IsTemporaryError(e3))

	assert.Equal(t, "wrap err arg1: some error: argErr", e1.Error())
	assert.Equal(t, "wrap err arg1: some error: argErr", e2.Error())
	assert.Equal(t, "wrap err arg1: some error: argErr", e3.Error())
}

func TestTemporaryErrorToLastError(t *testing.T) {
	t.Run("temporary error contains reason and component", func(t *testing.T) {
		// given
		tempErr1 := NewTemporaryError("Got status %d, component: %s, reason: %s", 890, string(ErrReconciler), string(ErrHttpStatusCode))
		expectMsg1 := fmt.Sprintf("Got status %d, component: %s, reason: %s", 890, string(ErrReconciler), string(ErrHttpStatusCode))

		tempErr2 := errors.Wrap(tempErr1, "something")
		expectMsg2 := "something: " + expectMsg1

		// when
		lastErr1 := ReasonForError(tempErr1)
		lastErr2 := ReasonForError(tempErr2)

		// then
		assert.Equal(t, ErrHttpStatusCode, lastErr1.Reason())
		assert.Equal(t, ErrReconciler, lastErr1.Component())
		assert.Equal(t, expectMsg1, lastErr1.Error())

		assert.Equal(t, ErrHttpStatusCode, lastErr2.Reason())
		assert.Equal(t, ErrReconciler, lastErr2.Component())
		assert.Equal(t, expectMsg2, lastErr2.Error())
	})

	t.Run("temporary error does not contain reason and component", func(t *testing.T) {
		// given
		tempErr1 := NewTemporaryError("Got status %d, component:, reason:", 890)
		expectMsg1 := fmt.Sprintf("Got status %d, component:, reason:", 890)

		tempErr2 := errors.Wrap(NewTemporaryError("temporary error..."), "something")
		expectMsg2 := "something: temporary error..."

		// when
		lastErr1 := ReasonForError(tempErr1)
		lastErr2 := ReasonForError(tempErr2)

		// then
		assert.Equal(t, ErrKEBInternal, lastErr1.Reason())
		assert.Equal(t, ErrKEB, lastErr1.Component())
		assert.Equal(t, expectMsg1, lastErr1.Error())

		assert.Equal(t, ErrKEBInternal, lastErr2.Reason())
		assert.Equal(t, ErrKEB, lastErr2.Component())
		assert.Equal(t, expectMsg2, lastErr2.Error())
	})
}
