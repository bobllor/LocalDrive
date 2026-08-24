package dbgateway

import (
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
)

// Error used for username validation failures.
var (
	UsernameEmptyErr            = errors.New("username cannot be empty")
	UsernameLenOutOfRangeErr    = fmt.Errorf("username must be between %d to %d characters long", USERNAME_MIN_LENGTH, USERNAME_MAX_LENGTH)
	UsernameInvalidFirstCharErr = errors.New("username must start with a letter")
	UsernameInvalidEndCharErr   = errors.New("username must end with an alphanumeric character")
	UsernameIsInvalidErr        = errors.New("username may only consist of alphanumeric characters and single periods")
	NoAffectedRowsErr           = errors.New("no rows were affected by the query")
)

// Error used for password validation faillures.
var (
	PasswordEmptyErr         = errors.New("password cannot be empty")
	PasswordNotEqualErr      = errors.New("passwords do not match")
	PasswordLenOutOfRangeErr = fmt.Errorf("password must be between %d to %d characters long", PASSWORD_MIN_LENGTH, PASSWORD_MAX_LENGTH)
)

// IsUsernameError compares if the given error is a username validation error.
func IsUsernameError(err error) bool {
	errs := []error{
		UsernameEmptyErr,
		UsernameInvalidEndCharErr,
		UsernameInvalidFirstCharErr,
		UsernameIsInvalidErr,
		UsernameLenOutOfRangeErr,
	}

	for _, e := range errs {
		if errors.Is(err, e) {
			return true
		}
	}

	return false
}

// IsPasswordError compares if the given error is a password validation error.
func IsPasswordError(err error) bool {
	errs := []error{
		PasswordEmptyErr,
		PasswordLenOutOfRangeErr,
		PasswordNotEqualErr,
	}

	for _, e := range errs {
		if errors.Is(err, e) {
			return true
		}
	}

	return false
}

// IsDuplicateSqlError checks if the error is a SQL error that is a duplicate
// entry error. This returns true if the error is a duplicate or a unique key
// entry error.
//
// If the error is not a duplicate error, or if it is not a SQL error, then
// it will be false.
func IsDuplicateSqlError(err error) bool {
	sqlErr, ok := err.(*mysql.MySQLError)
	duplicateEntry := uint16(1062)

	if !ok {
		return false
	}

	// TODO: need to figure out other errors with duplicate entries

	if sqlErr.Number == duplicateEntry {
		return true
	}

	return false
}
