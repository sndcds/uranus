package sql_utils

import (
	"fmt"
	"strings"
)

type UpdateBuilder struct {
	table      string
	setClauses []string
	args       []any
	argPos     int
	where      string
	whereArgs  []any
}

// NewUpdate creates a new builder
func NewUpdate(table string) *UpdateBuilder {
	return &UpdateBuilder{
		table:  table,
		argPos: 1,
	}
}

// Set adds a column = value (non-null only)
func (b *UpdateBuilder) Set(column string, value any) *UpdateBuilder {
	if value == nil {
		return b
	}

	b.setClauses = append(
		b.setClauses,
		fmt.Sprintf("%s = $%d", column, b.argPos),
	)

	b.args = append(b.args, value)
	b.argPos++
	return b
}

// SetNullable always sets value (including nil → SQL NULL)
func (b *UpdateBuilder) SetNullable(column string, value any) *UpdateBuilder {
	b.setClauses = append(
		b.setClauses,
		fmt.Sprintf("%s = $%d", column, b.argPos),
	)

	b.args = append(b.args, value)
	b.argPos++
	return b
}

// Where adds a WHERE clause (simple version)
func (b *UpdateBuilder) Where(condition string, args ...any) *UpdateBuilder {
	// NOTE: condition must already contain placeholders if needed
	b.where = condition
	b.whereArgs = append(b.whereArgs, args...)
	return b
}

// Build returns SQL + args
func (b *UpdateBuilder) Build() (string, []any, error) {
	if len(b.setClauses) == 0 {
		return "", nil, fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s",
		b.table,
		strings.Join(b.setClauses, ", "),
	)

	allArgs := append([]any{}, b.args...)

	// WHERE with separate args (shifted placeholders not handled here for simplicity)
	if b.where != "" {
		query += " WHERE " + b.where
		allArgs = append(allArgs, b.whereArgs...)
	}

	return query, allArgs, nil
}
