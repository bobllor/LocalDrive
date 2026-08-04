package sqlquery

import (
	"fmt"
	"strings"
)

// SqlSelect is used to perform SELECT queries.
type SqlSelect struct {
	// tableName is the table that the query is being performed on.
	TableName string

	// columns are the columns that are being selected. This
	// must have a value.
	columns []string

	// Args is any arguments used as the params in a query.
	Args []any

	// where is used to build WHERE clauses.
	// Initially this will be nil until the method Where is
	// explictly called in order to create the conditions.
	where *ConditionClause

	// queryBuilder is used to build the query.
	queryBuilder *QueryBuilder
}

// Select creates a new SqlSelect to create SELECT queries with
// the given columns on a table.
//
// If no columns are given, then it will default to selecting all columns.
func Select(tableName string, columns ...string) *SqlSelect {
	if len(columns) == 0 {
		columns = append(columns, "*")
	}

	return &SqlSelect{
		TableName:    tableName,
		columns:      columns,
		queryBuilder: &QueryBuilder{},
	}
}

// Build builds string for the base SELECT query.
func (s *SqlSelect) Build() (string, []any, error) {
	if s.where != nil {
		s.queryBuilder.FilterBuilder = s.where
	}
	query, err := s.queryBuilder.Build(s.buildQuery())

	return query, s.Args, err
}

// buildQuery builds the main query for the SELECT statement.
func (s *SqlSelect) buildQuery() string {
	mainElement := fmt.Sprintf(
		"%s %s FROM %s",
		SelectDML,
		strings.Join(s.columns, ","),
		s.TableName,
	)

	return mainElement
}

// Where creates a new WhereClause for conditional usage.
func (s *SqlSelect) Where() *ConditionClause {
	s.where = NewConditionClause(s, ConditionWhere)

	return s.where
}

func (s *SqlSelect) Write(args ...any) {
	s.Args = append(s.Args, args...)
}
