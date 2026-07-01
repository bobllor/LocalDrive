package sqlquery

import (
	"fmt"
	"strings"
)

type ConditionType string

const (
	ConditionWhere ConditionType = "WHERE"
)

// ConditionClause is a struct that is used to create condition clauses, supporting
// chaining and grouping of conditions.
//
// This is used in any queries that supports conditions.
type ConditionClause struct {
	// builder is a Builder interface that can build a query string.
	builder Builder

	// conditionType indicates what type of query the conditions are used for.
	// This is used to choose the correct type of clause for the conditions.
	// For example, WHERE or GROUP BY.
	conditionType ConditionType

	// node holds the group of conditions and its values.
	node *ConditionNode

	// head is the pointer to the first node in the ConditionClause. This should
	// never be modified or changed, as it is only used as a reference.
	head *ConditionNode

	// query is the full query of the ConditionClause built from the
	// node head when the Build method is used. This includes the conditionType.
	query string
}

// NewConditionClause creates a new ConditionClause.
func NewConditionClause(b Builder, conditionType ConditionType) *ConditionClause {
	return &ConditionClause{
		builder:       b,
		conditionType: conditionType,
	}
}

// Build builds the condition string for the condition clause.
func (c *ConditionClause) Build() (string, []any, error) {
	cQuery := c.buildConditionString()
	c.query = fmt.Sprintf("%s %s", c.conditionType, cQuery)

	query, args, err := c.builder.Build()

	return query, args, err
}

// Equal creates a new "equal to" condition. Example: `Column = value`.
func (c *ConditionClause) Equal(column string, arg any) *ConditionClause {
	c.addNewNode(column, OperatorEqual, arg)

	return c
}

// Is creates a new "IS" condition. Example: `Column IS NOT NULL`.
func (c *ConditionClause) Is(column string, arg string) *ConditionClause {
	c.addNewNode(column, OperatorIs, arg)

	return c
}

// In creates a new "in (list)" condition. Example: `Column IN (val1, val2...)`.
func (c *ConditionClause) In(column string, args ...any) *ConditionClause {
	c.addNewNode(column, OperatorIn, args...)

	return c
}

// Exists creates a new "exists (subquery)" condition. This is a special case in that
// an existing query must be passed along with any arguments it comes with.
//
// The arguments are appended to the c.Args.
func (c *ConditionClause) Exists(subquery string, args ...any) *ConditionClause {
	c.addNewNode(subquery, OperatorExists, args...)

	return c
}

// addNewNode adds a new node to the ConditionNode. This handles both initially nil values,
// and existing values with nodes to maintain the linked list chain.
//
// Arguments are also appended to c.args.
func (c *ConditionClause) addNewNode(column string, operator ComparisonOperator, args ...any) {
	node := NewConditionNode(column, operator, args...)
	if c.node == nil {
		c.node = node
		c.head = c.node
	} else {
		c.node.next = node
		c.node = c.node.next
	}

	c.builder.Write(args...)
}

// And sets the AND logical operator to the current node for use
// with the next node.
// If the current node is nil, it will do nothing.
func (c *ConditionClause) And() *ConditionClause {
	if c.node == nil {
		return c
	}

	c.node.NextLogicalOperator = OperatorAnd

	return c
}

// And sets the OR logical operator to the current node for use
// with the next node.
// If the current node is nil, it will do nothing.
func (c *ConditionClause) Or() *ConditionClause {
	if c.node == nil {
		return c
	}

	c.node.NextLogicalOperator = OperatorOr

	return c
}

// String creates a string representation of the ConditionNode data.
func (c *ConditionClause) String() string {
	temp := c.head

	out := []string{}

	for temp != nil {
		s := fmt.Sprintf(
			"[column:'%s',args:%v,operator:'%v',hasNext:%v]",
			temp.column,
			temp.args,
			temp.operator,
			temp.next != nil,
		)

		temp = temp.next
		out = append(out, s)
	}

	return strings.Join(out, ";")
}

// buildConditionString builds the condition string by accessing the nodes.
func (c *ConditionClause) buildConditionString() string {
	temp := c.head

	out := []string{}

	for temp != nil {
		s := temp.Build()
		out = append(out, s)

		if temp.next != nil && string(temp.NextLogicalOperator) != "" {
			out = append(out, string(temp.NextLogicalOperator))
		}

		temp = temp.next
	}

	return strings.Join(out, " ")
}
