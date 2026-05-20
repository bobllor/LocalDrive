package dbgateway

import (
	"database/sql"
	"fmt"

	"github.com/bobllor/cloud-project/src/file"
	"github.com/go-sql-driver/mysql"
)

const (
	dbDriver = "mysql"
)

type Gateway struct {
	File    *FileGateway
	User    *UserGateway
	Session *SessionGateway
}

// NewGateway creates a new Gateway containing the database gateways.
func NewGateway(fg *FileGateway, ug *UserGateway, sg *SessionGateway) *Gateway {
	gw := &Gateway{
		File:    fg,
		User:    ug,
		Session: sg,
	}

	return gw
}

// NewDatabase opens the SQL database and returns a sql.DB.
// It will return an error if any errors occur.
//
// The database is pinged in this call, if it fails then
// an error will be returned.
func NewDatabase(config *mysql.Config) (*sql.DB, error) {
	db, err := sql.Open(dbDriver, config.FormatDSN())
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

// NewConfig creates a new *mysql.Config from the arguments.
func NewConfig(user string, passwd string, net string, addr string, dbName string) *mysql.Config {
	c := mysql.NewConfig()

	c.User = user
	c.Passwd = passwd
	c.Net = net
	c.Addr = addr
	c.DBName = dbName

	c.AllowNativePasswords = true
	c.ParseTime = true

	return c
}

// exec executes a query on a database and returns the Result. This
// is only used for INSERT and UPDATE.
func execQuery(db *sql.DB, query string, args ...any) (sql.Result, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %v", err)
	}

	res, err := tx.Exec(query, args...)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to commit transaction for %s: %v", file.TableName, err)
	}

	return res, err
}
