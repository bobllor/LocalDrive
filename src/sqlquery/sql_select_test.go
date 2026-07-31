package sqlquery

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bobllor/assert"
	"github.com/bobllor/cloud-project/src/file"
	"github.com/bobllor/cloud-project/src/session"
	"github.com/bobllor/cloud-project/src/user"
)

func TestSelectBuild(t *testing.T) {
	query, _, err := Select(user.TableName, user.ColumnAccountID, user.ColumnActive, user.ColumnCreatedOn).
		Build()
	assert.Nil(t, err)

	baseQuery := fmt.Sprintf(
		"SELECT %s,%s,%s FROM %s",
		user.ColumnAccountID,
		user.ColumnActive,
		user.ColumnCreatedOn,
		user.TableName,
	)

	assert.Equal(t, query, baseQuery)
}

func TestSelectWhereBuild(t *testing.T) {
	t.Run("Basic Chaining", func(t *testing.T) {
		query, args, err := Select(user.TableName, user.ColumnUsername).Where().
			Equal(user.ColumnAccountID, "12345").And().
			Equal(user.ColumnActive, true).Or().
			Equal(user.ColumnCreatedOn, 12345).Build()
		assert.Nil(t, err)

		baseStr := fmt.Sprintf(
			"SELECT %s FROM %s WHERE %s = (?) AND %s = (?) OR %s = (?)",
			user.ColumnUsername,
			user.TableName,
			user.ColumnAccountID,
			user.ColumnActive,
			user.ColumnCreatedOn,
		)

		assert.Equal(t, query, baseStr)
		assert.Equal(t, len(args), 3)
	})

	t.Run("In Operator", func(t *testing.T) {
		query, args, err := Select(user.TableName, user.ColumnAccountID).Where().
			In(user.ColumnAccountID, "12345", "5678", "1111").Build()
		assert.Nil(t, err)

		baseStr := fmt.Sprintf(
			"SELECT %s FROM %s WHERE %s IN (?,?,?)",
			user.ColumnAccountID,
			user.TableName,
			user.ColumnAccountID,
		)

		assert.Equal(t, query, baseStr)
		assert.Equal(t, len(args), 3)
	})
}

func TestSelectWhereExistsSubquery(t *testing.T) {
	subSelect := Select(session.TableName, session.ColumnCreatedOn)
	subQ, subArgs, err := subSelect.Where().Equal(session.ColumnAccountID, "15555").Build()
	assert.Nil(t, err)
	mainQ, args, err := Select(user.TableName, user.ColumnAccountID).Where().
		Exists(subQ, subArgs...).Build()
	assert.Nil(t, err)

	baseStr := fmt.Sprintf(
		"SELECT %s FROM %s WHERE EXISTS (SELECT %s FROM %s WHERE %s = (?))",
		user.ColumnAccountID,
		user.TableName,
		session.ColumnCreatedOn,
		session.TableName,
		session.ColumnAccountID,
	)

	assert.Equal(t, mainQ, baseStr)
	assert.Equal(t, len(args), 1)
}

func TestSelectISQuery(t *testing.T) {
	q, _, err := Select(file.TableName).Where().Is(file.ColumnDeletedOn, "NOT NULL").Build()
	assert.Nil(t, err)

	assert.True(t, strings.Contains(q, fmt.Sprintf("WHERE %s IS NOT NULL", file.ColumnDeletedOn)))
}
