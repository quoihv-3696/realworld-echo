package repository

import "errors"

var ErrNotFound = errors.New("không tìm thấy bản ghi")

type DuplicateError struct {
	Field string
}

func (e *DuplicateError) Error() string {
	return "trùng giá trị ở cột " + e.Field
}
