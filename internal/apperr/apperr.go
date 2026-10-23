package apperr

import (
	"fmt"
	"net/http"
)

type Error struct {
	Status int
	Fields map[string][]string
}

func New(status int, field, message string) *Error {
	return &Error{Status: status, Fields: map[string][]string{field: {message}}}
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d %v", e.Status, e.Fields)
}

func (e *Error) StatusCode() int {
	return e.Status
}

func NotFound(field string) *Error {
	return New(http.StatusNotFound, field, "not found")
}

func Forbidden(field string) *Error {
	return New(http.StatusForbidden, field, "forbidden")
}

func Blank(field string) *Error {
	return New(http.StatusUnprocessableEntity, field, "can't be blank")
}

func Taken(field string) *Error {
	return New(http.StatusConflict, field, "has already been taken")
}

func TokenMissing() *Error {
	return New(http.StatusUnauthorized, "token", "is missing")
}

func InvalidCredentials() *Error {
	return New(http.StatusUnauthorized, "credentials", "invalid")
}

func TokenInvalid() *Error {
	return New(http.StatusUnauthorized, "token", "is invalid")
}

func TooShort(field string, min int) *Error {
	return New(http.StatusUnprocessableEntity, field, fmt.Sprintf("is too short (minimum is %d characters)", min))
}

func TooLong(field string, max int) *Error {
	return New(http.StatusUnprocessableEntity, field, fmt.Sprintf("is too long (maximum is %d characters)", max))
}

func NotNull(field string) *Error {
	return New(http.StatusUnprocessableEntity, field, "can't be null")
}
