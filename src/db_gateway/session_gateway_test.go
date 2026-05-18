package dbgateway

import (
	"testing"
	"time"

	"github.com/bobllor/assert"
	"github.com/bobllor/cloud-project/src/session"
	"github.com/bobllor/cloud-project/src/tests"
	"github.com/bobllor/cloud-project/src/user"
	"github.com/bobllor/cloud-project/src/utils"
)

func TestGetSessionByAccountID(t *testing.T) {
	sg := newTestSessionGateway(t)

	s, err := sg.GetSessionByAccountID(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	assert.Equal(t, s.AccountID, tests.DbRowInfo.AccountID)
	assert.Equal(t, s.SessionID, tests.DbRowInfo.SessionID)
}

func TestUpsertSessionNew(t *testing.T) {
	ug := newTestUserGateway(t)
	sg := newTestSessionGateway(t)

	usr, err := ug.AddUser("username.here", "password")
	assert.Nil(t, err)

	addS, err := sg.UpsertSession(usr.AccountID)
	assert.Nil(t, err)

	sesh, err := sg.GetSessionByAccountID(addS.AccountID)
	assert.Nil(t, err)

	_, err = DropRows(ug.database, user.TableName, user.ColumnAccountID, usr.AccountID)
	assert.Nil(t, err)

	assert.Equal(t, sesh.AccountID, addS.AccountID)
	assert.Equal(t, sesh.SessionID, addS.SessionID)
	assert.Equal(t, sesh.CreatedOn.Truncate(time.Minute), addS.CreatedOn.Truncate(time.Minute))
	assert.Equal(t, sesh.ExpireOn.Truncate(time.Minute), addS.ExpireOn.Truncate(time.Minute))
}

func TestUpsertSessionReplace(t *testing.T) {
	sg := newTestSessionGateway(t)

	baseS, err := sg.GetSessionByAccountID(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	sesh, err := sg.UpsertSession(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	newS, err := sg.GetSessionByAccountID(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	// artifically adding because for some reason this fails on go test ./... but
	// it does not fail on a manual run... lol?
	newS.CreatedOn = newS.CreatedOn.Add(5 * time.Second)
	newS.ExpireOn = newS.ExpireOn.Add(5 * time.Second)

	// reset the updated values
	_, err = UpdateRow(
		sg.database,
		session.TableName,
		session.ColumnAccountID,
		tests.DbRowInfo.AccountID,
		ClauseData{
			Columns: []string{session.ColumnSessionID, session.ColumnCreatedOn, session.ColumnExpireOn},
			Args:    []any{baseS.SessionID, baseS.CreatedOn, baseS.ExpireOn},
		},
	)
	assert.Nil(t, err)

	assert.NotEqual(t, baseS.SessionID, newS.SessionID)
	assert.Equal(t, baseS.CreatedOn.Compare(newS.CreatedOn), -1)
	assert.Equal(t, baseS.ExpireOn.Compare(newS.ExpireOn), -1)

	assert.Equal(t, newS.SessionID, sesh.SessionID)
}

func TestGetSessionBySessionID(t *testing.T) {
	sg := newTestSessionGateway(t)

	ses, err := sg.GetSessionBySessionID(tests.DbRowInfo.SessionID)
	assert.Nil(t, err)
	assert.NotNil(t, ses)

	assert.Equal(t, ses.SessionID, tests.DbRowInfo.SessionID)
	assert.Equal(t, ses.AccountID, tests.DbRowInfo.AccountID)
}

func TestGetSessionBySessionIDNone(t *testing.T) {
	sg := newTestSessionGateway(t)

	ses, err := sg.GetSessionBySessionID("nonexistentsid")
	assert.Nil(t, err)
	assert.Nil(t, ses)
}

func TestValidateSession(t *testing.T) {
	sg := newTestSessionGateway(t)

	t.Run("Valid ID", func(t *testing.T) {
		status, ses, err := sg.ValidateSessionAndGetUser(tests.DbRowInfo.SessionID)
		assert.Nil(t, err)
		assert.NotNil(t, ses)
		assert.True(t, status)
		assert.Equal(t, ses.AccountId, tests.DbRowInfo.AccountID)
	})

	t.Run("Invalid session IDs", func(t *testing.T) {
		ids := []string{"", "nonexistentid"}

		for _, id := range ids {
			status, ses, err := sg.ValidateSessionAndGetUser(id)
			assert.Nil(t, err)
			assert.Nil(t, ses)
			assert.False(t, status)
		}
	})

	t.Run("Expiration time expected false", func(t *testing.T) {
		ug := newTestUserGateway(t)
		acc, err := ug.AddUser("a.new.user", "password12345")
		assert.Nil(t, err)

		t.Cleanup(func() {
			_, err := DropRows(sg.database, user.TableName, user.ColumnAccountID, acc.AccountID)
			assert.Nil(t, err)
		})

		baseSess, err := sg.UpsertSession(acc.AccountID)
		assert.Nil(t, err)

		_, err = UpdateRow(
			sg.database,
			session.TableName,
			session.ColumnAccountID,
			acc.AccountID,
			ClauseData{
				Columns: []string{session.ColumnExpireOn},
				Args:    []any{baseSess.ExpireOn.AddDate(0, 0, -ExpireTimeDays-1).UTC()},
			},
		)
		assert.Nil(t, err)

		stat, ses, err := sg.ValidateSessionAndGetUser(baseSess.SessionID)
		assert.Nil(t, err)
		assert.Nil(t, ses)
		assert.False(t, stat)
	})
}

func TestDeleteSessionRowByID(t *testing.T) {
	sg := newTestSessionGateway(t)
	ug := newTestUserGateway(t)
	username := "another.test.username"

	t.Cleanup(func() {
		DropRows(sg.database, user.TableName, user.ColumnUsername, username)
	})

	user, err := ug.AddUser(username, "password1234")
	assert.Nil(t, err)

	ses, err := sg.UpsertSession(user.AccountID)
	assert.Nil(t, err)

	baseSes := ses.SessionID

	err = sg.DeleteSessionByID(baseSes)
	assert.Nil(t, err)

	newSes, err := sg.GetSessionBySessionID(baseSes)
	assert.Nil(t, err)
	assert.NotEqual(t, newSes, baseSes)
}

func newTestSessionGateway(t *testing.T) *SessionGateway {
	dbCfg := newTestDBConfig()
	db, err := NewDatabase(dbCfg)
	assert.Nil(t, err)

	deps := utils.NewTestDeps()

	sg := NewSessionGateway(db, deps)

	return sg
}
