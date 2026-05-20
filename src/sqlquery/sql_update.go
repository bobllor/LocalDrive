package sqlquery

import "fmt"

type SqlUpdate struct {
	// tableName is the table that the query is being performed on.
	TableName string

	// columns are the columns that are being updated. This
	// must have a value.
	columns []string

	// Args is any arguments used as the params in a query.
	// This must be the same length as columns.
	args []any

	// where is used to build WHERE conditionals.
	// Initially this will be nil until the method Where is
	// explictly called in order to create the conditions.
	where *ConditionClause

	// queryBuilder is used to build the query.
	queryBuilder *QueryBuilder
}

func Update(tableName string, columns ...string) *SqlArgs[*SqlUpdate] {
	update := &SqlUpdate{
		TableName:    tableName,
		columns:      columns,
		queryBuilder: &QueryBuilder{},
	}

	return &SqlArgs[*SqlUpdate]{
		builder: update,
	}
}

func (u *SqlUpdate) Where() *ConditionClause {
	u.where = NewConditionClause(u, ConditionWhere)

	return u.where
}

func (u *SqlUpdate) Write(args ...any) {
	u.args = append(u.args, args...)
}

func (u *SqlUpdate) Build() (string, []any, error) {
	setQ := BuildSetPlaceholder(u.columns)
	mainQ := fmt.Sprintf(
		"UPDATE %s SET %s",
		u.TableName,
		setQ,
	)
	u.queryBuilder.FilterBuilder = u.where

	query, err := u.queryBuilder.Build(mainQ)
	if err != nil {
		return "", nil, err
	}

	return query, u.args, nil
}
