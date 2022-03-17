package error

import (
	"fmt"

	"github.com/pkg/errors"
)

type TemporaryError struct {
	message string
}

func NewTemporaryError(msg string, args ...interface{}) *TemporaryError {
	return &TemporaryError{message: fmt.Sprintf(msg, args...)}
}

func AsTemporaryError(err error, context string, args ...interface{}) *TemporaryError {
	errCtx := fmt.Sprintf(context, args...)
	msg := fmt.Sprintf("%s: %s", errCtx, err.Error())

	return &TemporaryError{message: msg}
}

func (te TemporaryError) Error() string        { return te.message }
func (TemporaryError) Temporary() bool         { return true }
func (TemporaryError) Reason() ErrReason       { return ErrKEBInternal }
func (TemporaryError) Component() ErrComponent { return ErrKEB }

func IsTemporaryError(err error) bool {
	cause := errors.Cause(err)
	nfe, ok := cause.(interface {
		Temporary() bool
	})
	return ok && nfe.Temporary()
}

// can be used for temporary error
// but still storing the original error in case returned to Execute
type WrapTemporaryError struct {
	err error
}

func WrapAsTemporaryError(err error, msg string, args ...interface{}) *WrapTemporaryError {
	return &WrapTemporaryError{err: errors.Wrapf(err, msg, args...)}
}

func WrapNewTemporaryError(err error) *WrapTemporaryError {
	return &WrapTemporaryError{err: err}
}

func (te WrapTemporaryError) Error() string { return te.err.Error() }
func (WrapTemporaryError) Temporary() bool  { return true }

func (wte WrapTemporaryError) Reason() ErrReason {
	return ReasonForError(wte.err).Reason()
}

func (wte WrapTemporaryError) Component() ErrComponent {
	return ReasonForError(wte.err).Component()
}
