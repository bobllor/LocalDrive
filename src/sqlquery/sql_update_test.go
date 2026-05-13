package sqlquery

import (
	"testing"
	"time"

	"github.com/bobllor/assert"
	"github.com/bobllor/cloud-project/src/file"
	"github.com/google/uuid"
)

func TestUpdateBuild(t *testing.T) {
	fileName := "test1234.1010.log"
	modifiedOn := time.Now()
	fileId := uuid.New().String()

	sql := Update(file.TableName, file.ColumnFileName, file.ColumnModifiedOn).
		Args(fileName, modifiedOn).Where().Equal(file.ColumnFileID, fileId)

	q, args, err := sql.Build()
	assert.Nil(t, err)

	assert.Equal(t, len(args), 3)
	assert.Contains(t, q, file.TableName)
	assert.Contains(t, q, file.ColumnFileName)
	assert.Contains(t, q, file.ColumnModifiedOn)
	assert.Contains(t, q, file.ColumnFileID)
}
