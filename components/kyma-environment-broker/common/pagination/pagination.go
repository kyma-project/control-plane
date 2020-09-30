package pagination

import (
	"fmt"

	"github.com/pkg/errors"
)

func ConvertPagePageSizeAndOrderedColumnToSQL(pageSize, page int, orderedColumn string) (string, error) {
	if page < 1 {
		return "", errors.New("page cannot be smaller than 0")
	}

	if pageSize < 1 {
		return "", errors.New("page size cannot be smaller than 1")
	}

	return fmt.Sprintf(`ORDER BY %s LIMIT %d OFFSET %d`, orderedColumn, pageSize, (page-1)*pageSize), nil
}

func ConvertPageAndPageSizeToOffset(pageSize, page int) int {

	if page < 2 {
		return 0
	} else {
		return page*pageSize - 1
	}
}
