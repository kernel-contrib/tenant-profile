package internal

import (
	"errors"

	"gorm.io/gorm"
)

// IsNotFoundErr checks whether the error chain contains gorm.ErrRecordNotFound.
func IsNotFoundErr(err error) bool {
	return err != nil && errors.Is(err, gorm.ErrRecordNotFound)
}
