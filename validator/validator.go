package validator

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// santinel - ошибки
var (
	ErrEmptyTitle   = errors.New("title cannot be empty")
	ErrTitleTooLong = errors.New("title is too long")
	ErrPrice        = errors.New("price must be positive")
	ErrStock        = errors.New("stock must be non-negative")
)

func CheckValiation(title string, price float64, stock int) error {
	if strings.TrimSpace(title) == "" {
		return ErrEmptyTitle
	}
	if utf8.RuneCountInString(title) > 200 { // в руны безопаснее так как некоторые символы больше 1 байта
		return ErrTitleTooLong
	}
	if price <= 0 {
		return ErrPrice
	}
	if stock < 0 {
		return ErrStock
	}
	return nil
}
