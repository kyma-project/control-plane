package pagination

import (
	"fmt"

	"github.com/pkg/errors"
)

type Page struct {
	Count       int
	HasNextPage bool
}

func ConvertPageLimitAndOrderedColumnToSQL(pageSize, page int, orderedColumn string) (string, error) {
	if page < 1 {
		return "", errors.New("page cannot be smaller than 0")
	}

	if pageSize < 1 {
		return "", errors.New("page size cannot be smaller than 1")
	}

	return fmt.Sprintf(`ORDER BY %s LIMIT %d OFFSET %d`, orderedColumn, pageSize, (page-1)*pageSize), nil
}
