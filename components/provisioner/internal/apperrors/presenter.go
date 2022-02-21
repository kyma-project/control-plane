package apperrors

import (
	"context"
	"errors"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	log "github.com/sirupsen/logrus"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type presenter struct {
	Logger *log.Logger
}

func NewPresenter(logger *log.Logger) *presenter {
	return &presenter{Logger: logger}
}

func (p *presenter) Do(ctx context.Context, err error) *gqlerror.Error {
	customErr := AppError(nil)
	if ok := errors.As(err, &customErr); !ok {
		p.Logger.Errorf("Unknown error: %s\n", err.Error())
		return newGraphqlErrorResponse(ctx, CodeInternal, ErrProvisionerInternal, ErrProvisioner, err.Error())
	}

	if customErr.Code() == CodeInternal {
		p.Logger.Errorf("Internal Server Error: %s", err.Error())
	}

	return newGraphqlErrorResponse(ctx, customErr.Code(), customErr.Reason(), customErr.Component(), customErr.Error())
}

func newGraphqlErrorResponse(ctx context.Context, code ErrCode, reason ErrReason, component ErrComponent, msg string, args ...interface{}) *gqlerror.Error {
	return &gqlerror.Error{
		Message: fmt.Sprintf(msg, args...),
		Path:    graphql.GetResolverContext(ctx).Path(),
		Extensions: map[string]interface{}{
			"error_component": component,
			"error_reason":    reason,
			"error_code":      code,
		},
	}
}
