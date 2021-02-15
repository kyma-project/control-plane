package error

import (
	"fmt"

	"github.com/pkg/errors"
)

type NotValidFormatError struct {
	message string
}

func NewNotValidFormatError(msg string, args ...interface{}) *NotValidFormatError {
	return &NotValidFormatError{message: fmt.Sprintf(msg, args...)}
}

func AsNotValidFormatError(err error, context string, args ...interface{}) *NotValidFormatError {
	errCtx := fmt.Sprintf(context, args...)
	msg := fmt.Sprintf("%s: %s", errCtx, err.Error())

	return &NotValidFormatError{message: msg}
}

func (te NotValidFormatError) Error() string     { return te.message }
func (NotValidFormatError) NotValidFormat() bool { return true }

func IsNotValidFormatError(err error) bool {
	cause := errors.Cause(err)
	nfe, ok := cause.(interface {
		NotValidFormat() bool
	})
	return ok && nfe.NotValidFormat()
}
