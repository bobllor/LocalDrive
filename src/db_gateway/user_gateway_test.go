package dbgateway

import (
	"errors"
	"testing"

	"github.com/bobllor/assert"
	"github.com/bobllor/cloud-project/src/hasher"
	"github.com/bobllor/cloud-project/src/tests"
	"github.com/bobllor/cloud-project/src/user"
	"github.com/bobllor/cloud-project/src/utils"
)

const (
	testPassword = "anothertestpassword"
)

func TestGetUserByID(t *testing.T) {
	udb := newTestUserGateway(t)

	user, err := udb.GetUserByID(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	assert.Equal(t, user.AccountID, tests.DbRowInfo.AccountID)
	assert.True(t, user.Active)
	assert.Equal(t, user.Username, "test.username")

	_, err = hasher.ParsePHC(user.PasswordHash)
	assert.Nil(t, err)
}

func TestGetUserByUsername(t *testing.T) {
	ugw := newTestUserGateway(t)

	user, err := ugw.GetUserByUsername(tests.DbRowInfo.Username)
	assert.Nil(t, err)

	assert.Equal(t, user.Username, tests.DbRowInfo.Username)
	assert.Equal(t, user.PasswordHash, tests.DbRowInfo.PhcString)
}

func TestCheckCredentials(t *testing.T) {
	ugw := newTestUserGateway(t)

	status, _, err := ugw.ValidateUser(tests.DbRowInfo.Username, testPassword)
	assert.Nil(t, err)

	assert.True(t, status)
}

func TestCheckCredentialsInvalid(t *testing.T) {
	ugw := newTestUserGateway(t)

	status, ua, err := ugw.ValidateUser("userdoesnotexist", testPassword)
	assert.Nil(t, err)
	assert.False(t, status)
	assert.Nil(t, ua)

	status, _, err = ugw.ValidateUser(tests.DbRowInfo.Username, "invalidpassword")
	assert.Nil(t, err)
	assert.False(t, status)
}

func TestAddUser(t *testing.T) {
	ugw := newTestUserGateway(t)
	username := "auser.here"
	password := "somepasswordhere"

	acc, err := ugw.AddUser(username, password)
	assert.Nil(t, err)

	_, err = DropRows(ugw.database, user.TableName, user.ColumnAccountID, acc.AccountID)
	assert.Nil(t, err)
}

func TestAddUserComparePassword(t *testing.T) {
	ugw := newTestUserGateway(t)
	username := "a.userhere"
	password := "somepasswordhere"

	acc, err := ugw.AddUser(username, password)
	assert.Nil(t, err)

	// drop row immediately in case of failures below, the rest doesnt need the table data
	_, err = DropRows(ugw.database, user.TableName, user.ColumnAccountID, acc.AccountID)
	assert.Nil(t, err)

	baseRes, err := hasher.ParsePHC(acc.PasswordHash)
	assert.Nil(t, err)

	baseSalt, err := baseRes.DecodeSalt()
	assert.Nil(t, err)

	raw, err := hasher.Hash(password, baseSalt, baseRes.Params)
	assert.Nil(t, err)

	compareRes := raw.Encode()

	assert.Equal(t, compareRes.Hash, baseRes.Hash)
}

func TestDeleteUser(t *testing.T) {
	ugw := newTestUserGateway(t)

	err := ugw.DeleteUserByID(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	uInfo, err := ugw.GetUserByID(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	_, err = UpdateRow(
		ugw.database,
		user.TableName,
		user.ColumnAccountID,
		tests.DbRowInfo.AccountID,
		[]string{user.ColumnActive},
		true,
	)

	assert.Equal(t, uInfo.Active, false)
	assert.NotEqual(t, uInfo.Active, tests.DbRowInfo.UserActive)
}

func TestGetUserBySessionID(t *testing.T) {
	ug := newTestUserGateway(t)

	t.Run("Existing User", func(t *testing.T) {
		ua, err := ug.GetUserBySessionID(tests.DbRowInfo.SessionID)
		assert.Nil(t, err)
		assert.NotNil(t, ua)
		assert.Equal(t, ua.AccountID, tests.DbRowInfo.AccountID)
	})

	t.Run("User Does Not Exist", func(t *testing.T) {
		ua, err := ug.GetUserBySessionID("doesn't exist")
		assert.Nil(t, err)
		assert.Nil(t, ua)
	})
}

func TestValidateUsername(t *testing.T) {
	cases := []string{
		"abcdef.1",
		"a234567",
		"lolxdf",
		"usernamegoeshere",
		"username.goes.here",
	}

	for _, c := range cases {
		err := newTestUserGateway(t).validateUsername(c)

		assert.Nil(t, err)
	}
}

func TestValidateUsernameError(t *testing.T) {
	type cases struct {
		Value string
		Error error
	}
	testCases := []cases{
		{
			Value: "12345a",
			Error: UsernameInvalidFirstCharErr,
		},
		{
			Value: "_12345a",
			Error: UsernameInvalidFirstCharErr,
		},
		{
			Value: "asd fgh",
			Error: UsernameIsInvalidErr,
		},
		{
			Value: "asdf",
			Error: UsernameLenOutOfRangeErr,
		},
		{
			Value: "asdf][]h/;',.",
			Error: UsernameInvalidEndCharErr,
		},
		{
			Value: "abde..dsfd",
			Error: UsernameIsInvalidErr,
		},
		{
			Value: "abcde..sdfa.fff..s123",
			Error: UsernameIsInvalidErr,
		},
	}

	ug := newTestUserGateway(t)

	for _, c := range testCases {
		err := ug.validateUsername(c.Value)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, c.Error))
	}
}

func TestValidatePassword(t *testing.T) {
	type cases struct {
		Password        string
		ConfirmPassword string
		Error           error
	}

	ug := newTestUserGateway(t)
	tCases := []cases{
		{
			Password:        "abcdef!@#@",
			ConfirmPassword: "abcdef!@#@",
		},
		{
			Password:        "testPassWORD123$$!!",
			ConfirmPassword: "testPassWORD123$$!!",
		},
		{
			Password:        "wrongIncorrect",
			ConfirmPassword: "fdsa12345",
			Error:           PasswordNotEqualErr,
		},
		{
			Password:        "valu",
			ConfirmPassword: "valu",
			Error:           PasswordLenOutOfRangeErr,
		},
		{
			Password:        "",
			ConfirmPassword: "",
			Error:           PasswordEmptyErr,
		},
	}

	for _, c := range tCases {
		err := ug.validatePassword(c.Password, c.ConfirmPassword)
		if c.Error != nil {
			assert.NotNil(t, err)
			assert.True(t, IsPasswordError(err))
		} else {
			assert.Nil(t, err)
		}
	}
}

func newTestUserGateway(t *testing.T) *UserGateway {
	cnf := newTestDBConfig()
	db, err := NewDatabase(cnf)
	assert.Nil(t, err)

	deps := utils.NewTestDeps()

	ug := NewUserGateway(db, deps)

	return ug
}
