package dbgateway

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bobllor/cloud-project/src/hasher"
	"github.com/bobllor/cloud-project/src/session"
	"github.com/bobllor/cloud-project/src/sqlquery"
	"github.com/bobllor/cloud-project/src/user"
	"github.com/bobllor/cloud-project/src/utils"
	"github.com/google/uuid"
)

type UserGateway struct {
	database       *sql.DB
	userFieldCount int
	deps           *utils.Deps
}

func NewUserGateway(db *sql.DB, deps *utils.Deps) *UserGateway {
	return &UserGateway{
		database:       db,
		userFieldCount: user.ColumnSize,
		deps:           deps,
	}
}

// AddUser adds a new user into the database. It will return the UserAccount
// that was created in the database, or an error if one occurred.
//
// The username will be lowercased and spaces are trimmed by default.
// The password is stored as the PHC string from the password hashing function.
//
// If the username and password fails to validate, it will return an error that is one of
// password or username validation error. There are helper functions that determine the error
// type.
// Generic errors are returned if an unexpected error occurred during normal processing.
func (ug *UserGateway) AddUser(username string, password string) (*user.UserAccount, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	accountID := uuid.NewString()
	raw, err := hasher.Hash(password, nil, hasher.DefaultArgon2Params)
	if err != nil {
		ug.deps.Log.Criticalf("Failed to hash password: %v", err)
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	err = ug.validateUsername(username)
	if err != nil {
		if IsUsernameError(err) {
			ug.deps.Log.Infof("Failed to validate username: %v", err)
			// the error is used to display on the frontend
			return nil, err
		} else {
			ug.deps.Log.Criticalf("Username validation had an error: %v", err)
			return nil, fmt.Errorf("an unknown error occurred")
		}
	}

	hashRes := raw.Encode()

	acc := &user.UserAccount{
		AccountID:    accountID,
		PasswordHash: hashRes.PHC,
		Username:     username,
		CreatedOn:    time.Now().UTC(),
		Active:       true,
	}

	args := acc.ToArgs()

	query, args, err := sqlquery.InsertInto(
		user.TableName,
		user.ColumnAccountID,
		user.ColumnUsername,
		user.ColumnPasswordHash,
		user.ColumnCreatedOn,
		user.ColumnActive,
	).Args(args...).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build INSERT INTO query: %v", err)
	}

	res, err := execQuery(ug.database, query, args...)
	if err != nil {
		ug.deps.Log.Warnf("Failed to execute query: %v | query: %s", err, query)
		return nil, err
	}

	ug.deps.Log.Infof("Successfully added user '%s'", username)
	logResultRows(ug.deps.Log, res)

	return acc, nil
}

const (
	USERNAME_MIN_LENGTH = 6
	USERNAME_MAX_LENGTH = 32
)

// validateUsername validates the username. If it fails to validate, it will
// return an error.
//
// Errors will be a password validation error or a generic error if the regex
// compile fails.
func (ug *UserGateway) validateUsername(username string) error {
	if strings.TrimSpace(username) == "" {
		return UsernameEmptyErr
	}

	if len(username) < USERNAME_MIN_LENGTH || len(username) > USERNAME_MAX_LENGTH {
		return UsernameLenOutOfRangeErr
	}

	firstChar := string(username[0])
	lastChar := string(username[len(username)-1])

	alphaOnlyRegex := "[A-Za-z]"
	alphaNumericRegex := "[A-Za-z0-9]"
	// used to prevent double periods in the username
	doublePeriodRegex := ".*[..]{2}.*"

	stat, err := regexp.MatchString(alphaOnlyRegex, firstChar)
	if err != nil {
		return fmt.Errorf("failed to compile regex: %v", err)
	}
	if !stat {
		return UsernameInvalidFirstCharErr
	}

	stat, err = regexp.MatchString(alphaNumericRegex, lastChar)
	if err != nil {
		return fmt.Errorf("failed to compile regex: %v", err)
	}
	if !stat {
		return UsernameInvalidEndCharErr
	}

	stat, err = regexp.MatchString(doublePeriodRegex, username)
	if err != nil {
		return fmt.Errorf("failed to compile regex: %v", err)
	}
	if stat {
		return UsernameIsInvalidErr
	}

	usernameRegex := `^([A-Za-z0-9.]+)$`
	stat, err = regexp.MatchString(usernameRegex, username)
	if err != nil {
		return fmt.Errorf("failed to compile regex: %v", err)
	}
	if !stat {
		return UsernameIsInvalidErr
	}

	return nil
}

const (
	PASSWORD_MIN_LENGTH = 8
	PASSWORD_MAX_LENGTH = 64
)

// validatePassword validates the password. If it fails to validate, it will
// return an error.
//
// Errors will be a password validation error or a generic error if the regex
// compile fails.
func (ug *UserGateway) validatePassword(pw string, confirmPw string) error {
	if pw == "" {
		return PasswordEmptyErr
	}

	if len(pw) < PASSWORD_MIN_LENGTH || len(pw) > PASSWORD_MAX_LENGTH {
		return PasswordLenOutOfRangeErr
	}

	if pw != confirmPw {
		return PasswordNotEqualErr
	}

	return nil
}

// GetUserByUsername gets the user row based on the username.
// If the user does not exist, then it will return nil.
func (ug *UserGateway) GetUserByUsername(username string) (*user.UserAccount, error) {
	query, args, err := sqlquery.Select(user.TableName).Where().Equal(user.ColumnUsername, username).Build()
	if err != nil {
		return nil, err
	}

	rows, err := ug.database.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v | query: %s", err, query)
	}

	users := []user.UserAccount{}

	err = SelectRows(rows, &users)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rows from query: %v", err)
	}

	if len(users) == 0 {
		ug.deps.Log.Infof("user %s has no entries", username)
		return nil, nil
	}

	return &users[0], nil
}

// ValidateUser validates if the credentials are correct for the user. The username and
// password is compared and will return a boolean and the user info. If an error occurs,
// then an error will be returned instead.
//
// The username will be lowercased and spaces are trimmed by default.
//
// If validation is true, then the user will always be returned.
func (ug *UserGateway) ValidateUser(username string, password string) (bool, *user.UserAccount, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	user, err := ug.GetUserByUsername(username)
	if err != nil {
		return false, nil, err
	}
	if user == nil {
		return false, nil, nil
	}

	storedHash, err := hasher.ParsePHC(user.PasswordHash)
	if err != nil {
		return false, nil, err
	}

	validCredentials, err := hasher.Compare(password, storedHash)
	if err != nil {
		return false, nil, err
	}

	return validCredentials, user, nil
}

// GetUserByID retrieves the user row based on the account ID.
func (ug *UserGateway) GetUserByID(accountID string) (*user.UserAccount, error) {
	query, args, err := sqlquery.Select(user.TableName).Where().Equal(user.ColumnAccountID, accountID).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build SELECT query: %v", err)
	}

	user := user.UserAccount{}

	rows, err := ug.database.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v | query: %s", err, query)
	}

	err = SelectRow(rows, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rows from query: %v", err)
	}

	return &user, nil
}

// GetUserBySessionID retrieves the user row based on the session ID. If the
// session ID is not found, then it will return nil.
// The return will contain all the columns except the password column of the row.
//
// If there is a session ID, then there will be an existing user. This does not apply vice versa.
func (ug *UserGateway) GetUserBySessionID(sessionID string) (*user.UserAccountNoPassword, error) {

	sQuery, sArgs, err := sqlquery.Select(session.TableName, session.ColumnAccountID).
		Where().Equal(session.ColumnSessionID, sessionID).Build()
	if err != nil {
		return nil, err
	}

	query, _, err := sqlquery.Select(
		user.TableName,
		user.ColumnAccountID,
		user.ColumnUsername,
		user.ColumnCreatedOn,
		user.ColumnActive,
	).Build()
	if err != nil {
		return nil, err
	}

	query = fmt.Sprintf(`
		%s 
		WHERE %s IN (%s)
		`,
		query,
		user.ColumnAccountID,
		sQuery,
	)

	rows, err := ug.database.Query(query, sArgs...)
	if err != nil {
		ug.deps.Log.Criticalf("Failed to execute query: %v | query: %s", err, query)
		return nil, err
	}

	var us []user.UserAccountNoPassword
	err = SelectRows(rows, &us)
	if err != nil {
		ug.deps.Log.Criticalf("Failed to parse SQL rows: %v", err)
		return nil, err
	}

	if len(us) == 0 {
		ug.deps.Log.Infof("No user found with provided session ID")
		return nil, nil
	}

	return &us[0], nil
}

// DeleteUserByID sets an account ID for deletion. This is a soft deletion,
// it sets the Active column to false for deletion at a later date.
func (ug *UserGateway) DeleteUserByID(accountID string) error {
	query, args, err := sqlquery.Update(user.TableName, user.ColumnActive).Args(false).
		Where().Equal(user.ColumnAccountID, accountID).Build()

	res, err := execQuery(ug.database, query, args...)
	if err != nil {
		return err
	}

	logResultRows(ug.deps.Log, res)

	return nil
}

// RestoreUserByID removes the soft deletion state from the account ID.
func (ug *UserGateway) RestoreUserByID(accountID string) error {
	return nil
}
