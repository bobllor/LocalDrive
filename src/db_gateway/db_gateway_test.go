package dbgateway

import (
	"github.com/go-sql-driver/mysql"
)

// newTestDBConfig creates a test DB config for use in test environments.
func newTestDBConfig() *mysql.Config {
	port := "3307"

	user := "root"
	password := ""
	net := "tcp"
	addr := "127.0.0.1" + ":" + port
	dbName := "TestLocalCloudStorage"

	dbConfig := NewConfig(user, password, net, addr, dbName)

	return dbConfig
}
