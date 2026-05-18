package dbgateway

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bobllor/cloud-project/src/session"
	"github.com/bobllor/cloud-project/src/sqlquery"
	"github.com/bobllor/cloud-project/src/user"
	"github.com/bobllor/cloud-project/src/utils"
	"github.com/google/uuid"
)

const (
	ExpireTimeDays = 14
)

func NewSessionGateway(db *sql.DB, deps *utils.Deps) *SessionGateway {
	sg := &SessionGateway{
		database:          db,
		sessionFieldCount: session.ColumnSize,
		deps:              deps,
	}

	return sg
}

type SessionGateway struct {
	database          *sql.DB
	sessionFieldCount int
	deps              *utils.Deps
}

type UserSessionInfo struct {
	SessionId string
	ExpireOn  time.Time
	Username  string
	AccountId string
}

// GetSessionByAccountID retrieves the session of the account ID. If a session is not found,
// then it will return nil.
func (sg *SessionGateway) GetSessionByAccountID(accountID string) (*session.Session, error) {
	accSession := []session.Session{}

	query, args, err := sqlquery.Select(session.TableName).Where().Equal(session.ColumnAccountID, accountID).Build()
	if err != nil {
		return nil, err
	}

	rows, err := sg.database.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %v | query: %s", err, query)
	}

	err = SelectRows(rows, &accSession)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database rows: %v", err)
	}

	if len(accSession) == 0 {
		return nil, nil
	}

	return &accSession[0], nil
}

// GetSessionBySessionID retrieves the session by session ID. If a session is not found,
// then it will return nil.
func (sg *SessionGateway) GetSessionBySessionID(sessionID string) (*session.Session, error) {
	if !sg.validateID(sessionID) {
		return nil, nil
	}

	query, args, err := sqlquery.Select(session.TableName).Where().Equal(session.ColumnSessionID, sessionID).Build()
	if err != nil {
		return nil, err
	}

	rows, err := sg.database.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %v | query: %s", err, query)
	}

	var ses []session.Session
	err = SelectRows(rows, &ses)
	if err != nil {
		return nil, err
	}

	if len(ses) == 0 {
		sg.deps.Log.Info("No session ID found")
		return nil, nil
	}

	return &ses[0], nil
}

// UpsertSession adds a new or updates an existing entry for the Session table,
// generating a new session ID associated with the account ID to maintain a session.
// It will return the Session that was added if successful.
//
// Existing account IDs will have get updated with new session ID
// dates, and if it is expired then a new session ID is generated.
func (sg *SessionGateway) UpsertSession(accountID string) (*session.Session, error) {
	sessionID := uuid.NewString()
	query := fmt.Sprintf("INSERT INTO %s", session.TableName)

	currTime := time.Now().UTC()
	expireTime := currTime.AddDate(0, 0, ExpireTimeDays)
	ses := session.Session{
		SessionID: sessionID,
		AccountID: accountID,
		CreatedOn: currTime,
		ExpireOn:  expireTime,
	}

	args := ses.ToArgs()

	placeholder := BuildPlaceholder(len(args), 1)

	duplicateStr := fmt.Sprintf(
		"ON DUPLICATE KEY UPDATE %s=?,%s=?,%s=?",
		session.ColumnSessionID,
		session.ColumnCreatedOn,
		session.ColumnExpireOn,
	)

	AppendArgs(&args, sessionID, currTime, expireTime)

	query = query + " " + "VALUES" + placeholder + " " + duplicateStr

	res, err := execQuery(sg.database, query, args...)
	if err != nil {
		sg.deps.Log.Warnf("Failed to execute query: %s | Values: %v", query, args)
		return nil, err
	}

	sg.deps.Log.Info("Successfully upsert session")
	logResultRows(sg.deps.Log, res)

	return &ses, nil
}

// ValidateSessionAndGetUser validates a session with the user's session ID and returns
// if the session is valid and a UserSessionInfo struct.
//
// If the session ID is invalid, it does not exist, it does not match the stored database
// version, or if it is expired, then it will return false.
//
// Any errors will be returned during database queries or parsing.
func (sg *SessionGateway) ValidateSessionAndGetUser(sessionID string) (bool, *UserSessionInfo, error) {
	// false conditions:
	//	- any DB errors (error must be handled)
	//	- sessionID are empty strings or invalid formatting
	//	- sessionID does not match stored sessionID
	//	- stored expiration date is < current time
	//	- session row is not found with account ID

	if !sg.validateID(sessionID) {
		return false, nil, nil
	}

	// TODO: add cache access here, probably redis or if you are lazy a hash map

	query := fmt.Sprintf(`
		SELECT s.%s, s.%s, u.%s, u.%s
		FROM %s s
		JOIN %s u
			ON u.%s = s.%s
		WHERE s.%s = ?
		LIMIT 1`,
		session.ColumnSessionID, session.ColumnExpireOn,
		user.ColumnUsername, user.ColumnAccountID,
		session.TableName, user.TableName,
		user.ColumnAccountID, session.ColumnAccountID,
		session.ColumnSessionID,
	)

	rows, err := sg.database.Query(query, sessionID)
	if err != nil {
		return false, nil, SqlErr
	}

	var userSesInfo []UserSessionInfo
	err = SelectRows(rows, &userSesInfo)
	if err != nil {
		return false, nil, SqlErr
	}
	if len(userSesInfo) == 0 {
		return false, nil, nil
	}
	ses := userSesInfo[0]
	if ses.SessionId != sessionID {
		return false, nil, nil
	}
	if ses.ExpireOn.UTC().Before(time.Now().UTC()) {
		return false, nil, nil
	}

	return true, &ses, nil
}

// DeleteSessionByID deletes the given session ID from the table.
//
// If the session ID does not exist in the server, then this will do nothing.
// Any errors that occur will be database related.
//
// This method does not remove the cookie from the client.
func (sg *SessionGateway) DeleteSessionByID(sessionID string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE %s = ?
		`,
		session.TableName,
		session.ColumnSessionID,
	)

	res, err := execQuery(sg.database, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete from Session table: %v", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		if n == 0 {
			sg.deps.Log.Info("No existing session ID found")
		}
	}

	sg.deps.Log.Info("Successfully invalidated session ID")

	return nil
}

// validateID validates the ID string formatting. It returns a true
// if it is valid, otherwise it will return false.
// This does not check the database.
func (sg *SessionGateway) validateID(id string) bool {
	if strings.TrimSpace(id) == "" {
		return false
	}

	return true
}
