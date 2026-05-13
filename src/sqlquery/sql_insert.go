package sqlquery

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// InsertInvalidArgCount is an error where the length of arguments is not a mulitple
	// of the length of columns. In other words, len(arguments) % len(columns) != 0.
	InsertInvalidArgCountErr = errors.New("arguments count must form complete rows of columns count")
	InsertZeroColumnsErr     = errors.New("columns cannot be of length 0")
)

// SqlInsert is used to build INSERT queries.
type SqlInsert struct {
	// tableName is the table that the query is being performed on.
	TableName string

	// columns are the columns that are being inserted into the
	// row.
	columns []string

	// args is any arguments used as the params in a query.
	args []any

	// queryBuilder is used to build the query.
	queryBuilder *QueryBuilder
}

// InsertInto creates a new INSERT INTO query struct. It uses a table name
// and any column arguments.
//
// It will return a SqlArgs for inserting arguments for each column.
func InsertInto(tableName string, columns ...string) *SqlArgs[*SqlInsert] {
	s := &SqlInsert{
		TableName:    tableName,
		columns:      columns,
		queryBuilder: &QueryBuilder{},
	}

	sargs := &SqlArgs[*SqlInsert]{
		builder: s,
	}

	return sargs
}

// Write adds the arguments used for the rows to insert into the table.
func (s *SqlInsert) Write(args ...any) {
	s.args = append(s.args, args...)
}

// Build creates the query for the INSERT statement.
// If no arguments were given into
func (s *SqlInsert) Build() (string, []any, error) {
	if len(s.args)%len(s.columns) != 0 {
		return "", nil, InsertInvalidArgCountErr
	}
	if len(s.columns) == 0 {
		return "", nil, InsertZeroColumnsErr
	}

	query, err := s.queryBuilder.Build(s.buildQuery())

	return query, s.args, err
}

// buildQuery builds the main query for the INSERT statement.
func (s *SqlInsert) buildQuery() string {
	// divides how many repeated params are needed for completed rows
	// s.args must be divisible by s.columns, or A % B == 0.
	repeatAmount := len(s.args) / len(s.columns)
	params := BuildPlaceholder(len(s.columns), repeatAmount)

	holder := []string{}
	mainQuery := fmt.Sprintf(
		"%s %s",
		InsertIntoDML,
		s.TableName,
	)

	columns := s.buildColumns()

	holder = append(holder, mainQuery)
	if columns != "" {
		holder = append(holder, columns)
	}
	holder = append(holder, fmt.Sprintf("VALUES %s", params))

	return strings.Join(holder, " ")
}

// buildColumns creates the columns based on s.columns.
// If s.columns is empty, then it will return an empty string.
func (s *SqlInsert) buildColumns() string {
	if len(s.columns) == 0 {
		return ""
	}

	holder := []string{}

	for _, column := range s.columns {
		holder = append(holder, column)
	}

	out := fmt.Sprintf(
		"(%s)",
		strings.Join(holder, ","),
	)

	return out
}
