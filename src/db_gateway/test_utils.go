package dbgateway

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/bobllor/assert"
	"github.com/bobllor/cloud-project/src/tests"
	"github.com/bobllor/cloud-project/src/utils"
)

type GatewayDBOptions struct {
	// CreateStorage is used to trigger the creation of the storage and
	// its files in the temp directory.
	CreateStorage bool
	CreateTemp    bool
}

// NewTestGateway creates a test Gateway and the test sql.DB for use. The Gateway will
// use t.TempDir for its absolute paths for writing, but creation of the paths will
// require the options to be given.
//
// An optional argument can be given to trigger effects in the function.
//
// If an error occurs, then it will fatal and exit.
func NewTestGatewayDB(t *testing.T, opts ...GatewayDBOptions) (*Gateway, *sql.DB) {
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

	var opt GatewayDBOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "testapp", "storage")
	tempPath := filepath.Join(tmpDir, "tmp")

	if opt.CreateStorage {
		// obtained from sql test db
		// NOTE: maybe should make it an easier to read ID but whatever.
		fileIds := []string{tests.DbRowInfo.FileID, "anotherfileidhere"}

		for _, id := range fileIds {
			filePath := filepath.Join(storagePath, tests.DbRowInfo.AccountID, id)

			err = os.MkdirAll(filepath.Dir(filePath), 0o777)
			assert.Nil(t, err)

			err = os.WriteFile(filePath, nil, 0o744)
			assert.Nil(t, err)
		}
	}
	if opt.CreateTemp {
		err = os.MkdirAll(tempPath, 0o700)
		assert.Nil(t, err)
	}

	fg := NewFileGateway(tdb, deps)
	ug := NewUserGateway(tdb, deps)
	sg := NewSessionGateway(tdb, deps)

	ds := DirectoryStore{
		Storage: storagePath,
		Temp:    tempPath,
	}

	gw := NewGateway(fg, ug, sg, ds)

	return gw, tdb
}
