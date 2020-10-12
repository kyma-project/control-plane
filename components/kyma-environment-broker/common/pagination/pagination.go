package pagination

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
)

func ConvertPageSizeAndOrderedColumnToSQL(pageSize, page int, orderedColumn string) (string, error) {
	err := ValidatePageParameters(pageSize, page)
	if err != nil {
		return "", errors.Wrap(err, "while validating page parameters")
	}

	return fmt.Sprintf(`ORDER BY %s LIMIT %d OFFSET %d`, orderedColumn, pageSize, (page-1)*pageSize), nil
}

func ValidatePageParameters(pageSize, page int) error {
	if page < 1 {
		return errors.New("page cannot be smaller than 1")
	}

	if pageSize < 1 {
		return errors.New("page size cannot be smaller than 1")
	}
	return nil
}

func ConvertPageAndPageSizeToOffset(pageSize, page int) int {
	if page < 2 {
		return 0
	} else {
		return page*pageSize - 1
	}
}

const (
	pageSizeParam = "page_size"
	pageParam     = "page"
)

func ExtractPaginationConfigFromRequest(req *http.Request, maxPage int) (int, int, error) {
	var pageSize int
	var page int
	var err error

	params := req.URL.Query()
	pageSizeArr, ok := params[pageSizeParam]
	if len(pageSizeArr) > 1 {
		return 0, 0, errors.New("pageSize has to be one parameter")
	}

	if !ok {
		pageSize = maxPage
	} else {
		pageSize, err = strconv.Atoi(pageSizeArr[0])
		if err != nil {
			return 0, 0, errors.New("pageSize has to be an integer")
		}
	}

	if pageSize > maxPage {
		return 0, 0, errors.New(fmt.Sprintf("pageSize is bigger than maxPage(%d)", maxPage))
	}

	pageArr, ok := params[pageParam]
	if len(pageArr) > 1 {
		return 0, 0, errors.New("page has to be one parameter")
	}
	if !ok {
		page = 1
	} else {
		page, err = strconv.Atoi(pageArr[0])
		if err != nil {
			return 0, 0, errors.New("page has to be an integer")
		}
	}

	return pageSize, page, nil
}
