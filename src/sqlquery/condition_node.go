package sqlquery

import "fmt"

// ConditionNode is a node representing a single condition. It contains an
// internal node pointer that is used to create the condition chain
// for a query.
type ConditionNode struct {
	// column is the column or subquery that is being used.
	column string

	// args is a slice of any type.
	args []any

	// operator is the operator used for conditionals.
	operator ComparisonOperator

	// next is the next ConditionNode for chaining.
	next *ConditionNode

	// NextLogicalOperator is the operator used for truth conditions for
	// the next node.
	NextLogicalOperator LogicalOperator
}

// NewConditionNode creates a new ConditionNode.
func NewConditionNode(column string, compOperator ComparisonOperator, args ...any) *ConditionNode {
	return &ConditionNode{
		column:   column,
		operator: compOperator,
		args:     args,
	}
}

// SetColumn sets the column value. If a column value is not needed,
// then an empty string can be set.
func (c *ConditionNode) SetColumn(column string) {
	c.column = column
}

// AppendArgs appends any args into the arg values.
func (c *ConditionNode) AppendArgs(args ...any) {
	for _, arg := range args {
		c.args = append(c.args, arg)
	}
}

// Build builds the query string for the node. This does not include the WHERE
// clause, it only contains the conditions.
func (c *ConditionNode) Build() string {
	query := c.buildQuery()

	return query
}

// buildQuery builds the query string for the node and handles
// operator specific strings.
func (c *ConditionNode) buildQuery() string {
	params := BuildPlaceholder(len(c.args), 1)
	var query string
	switch c.operator {
	case OperatorIn:
		query = fmt.Sprintf("%s IN %s", c.column, params)
	case OperatorEqual:
		query = fmt.Sprintf("%s = %s", c.column, params)
	case OperatorExists:
		// special case, the subquery is considered the column
		// args are not used here, args is added in the WhereClause
		query = fmt.Sprintf("EXISTS (%s)", c.column)
	case OperatorIs:
		// this has at most 0 arguments
		query = fmt.Sprintf("%s IS %v", c.column, c.args[0])
	default:
		// default case, this will fail a query normally which can be caught if
		// we dont have a proper case (or if its wrong).
		query = fmt.Sprintf("%s %s %s", c.column, string(c.operator), params)
	}

	return query
}
