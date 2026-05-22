package dbgateway

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/bobllor/assert"
	"github.com/bobllor/cloud-project/src/tests"
	"github.com/bobllor/cloud-project/src/utils"
)

// NewTestGateway creates a test Gateway and the test sql.DB for use.
// If an error occurs, then it will fatal and exit.
func NewTestGatewayDB(t *testing.T) (*Gateway, *sql.DB) {
	dbcfg := NewConfig(
		tests.DbConInfo.User,
		tests.DbConInfo.Password,
		tests.DbConInfo.Net,
		tests.DbConInfo.Addr,
		tests.DbConInfo.DbName,
	)

	tdb, err := NewDatabase(dbcfg)
	assert.Nil(t, err)

	deps := utils.NewTestDeps()

	storagePath := filepath.Join(t.TempDir(), "testapp")

	fg := NewFileGateway(tdb, deps)
	ug := NewUserGateway(tdb, deps)
	sg := NewSessionGateway(tdb, deps)

	gw := NewGateway(fg, ug, sg, storagePath)

	return gw, tdb
}
