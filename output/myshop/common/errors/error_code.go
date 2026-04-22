package errors

import "fmt"

const (
	ErrCodeSuccess = 0
	ErrCodeNotFound = 1002
)

type BusinessError struct {
	Code int `json:"code"`
	Message string `json:"message"`
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func NewBusinessError(code int, msg string) *BusinessError {
	return &BusinessError{Code: code, Message: msg}
}
