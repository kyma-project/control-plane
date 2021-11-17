package error

import  "github.com/pkg/errors"

type NotFoundError struct {

}

func (NotFoundError) Error() string {
	return "not found"
}

func (NotFoundError) IsNotFound() bool {
	return true
}

func IsNotFoundError(err error) bool {
	cause := errors.Cause(err)
	nfe, ok := cause.(interface {
		IsNotFound() bool
	})
	return ok && nfe.IsNotFound()
}