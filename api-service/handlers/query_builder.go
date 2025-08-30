package handlers

import (
	"fmt"
	"strings"
)

// QueryBuilder helps build safe SQL queries with parameterized values
type QueryBuilder struct {
	baseQuery string
	args      []interface{}
	argIndex  int
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(baseQuery string) *QueryBuilder {
	return &QueryBuilder{
		baseQuery: baseQuery,
		args:      []interface{}{},
		argIndex:  1,
	}
}

// AddWhereCondition adds a WHERE condition to the query
func (qb *QueryBuilder) AddWhereCondition(field string, value interface{}) {
	if !strings.Contains(qb.baseQuery, "WHERE") {
		qb.baseQuery += fmt.Sprintf(" WHERE %s = $%d", field, qb.argIndex)
	} else {
		qb.baseQuery += fmt.Sprintf(" AND %s = $%d", field, qb.argIndex)
	}
	qb.args = append(qb.args, value)
	qb.argIndex++
}

// AddOrderBy adds ORDER BY clause
func (qb *QueryBuilder) AddOrderBy(field, direction string) {
	// Validate direction to prevent injection
	direction = strings.ToUpper(direction)
	if direction != "ASC" && direction != "DESC" {
		direction = "DESC" // Default to DESC
	}
	
	// Basic validation for field name (should be alphanumeric + underscore)
	if !isValidFieldName(field) {
		field = "created_at" // Default safe field
	}
	
	qb.baseQuery += fmt.Sprintf(" ORDER BY %s %s", field, direction)
}

// AddLimitOffset adds LIMIT and OFFSET clauses
func (qb *QueryBuilder) AddLimitOffset(limit, offset int) {
	qb.baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", qb.argIndex, qb.argIndex+1)
	qb.args = append(qb.args, limit, offset)
	qb.argIndex += 2
}

// Build returns the final query and arguments
func (qb *QueryBuilder) Build() (string, []interface{}) {
	return qb.baseQuery, qb.args
}

// GetArgIndex returns the current argument index (useful for extending queries)
func (qb *QueryBuilder) GetArgIndex() int {
	return qb.argIndex
}

// GetArgs returns the current arguments
func (qb *QueryBuilder) GetArgs() []interface{} {
	return qb.args
}

// isValidFieldName checks if a field name is safe (alphanumeric + underscore only)
func isValidFieldName(field string) bool {
	if len(field) == 0 || len(field) > 50 {
		return false
	}
	
	for _, char := range field {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || 
			 char == '_') {
			return false
		}
	}
	return true
}

// RecipesQueryBuilder builds recipes queries safely
type RecipesQueryBuilder struct {
	*QueryBuilder
}

// NewRecipesQueryBuilder creates a new recipes query builder
func NewRecipesQueryBuilder() *RecipesQueryBuilder {
	baseQuery := `
		SELECT 
			id, title, servings, instructions, tips, status, user_id, created_at, updated_at,
			COUNT(*) OVER() as total_count
		FROM recipes`
	
	return &RecipesQueryBuilder{
		QueryBuilder: NewQueryBuilder(baseQuery),
	}
}

// WithStatus adds status filter
func (rqb *RecipesQueryBuilder) WithStatus(status string) *RecipesQueryBuilder {
	// Validate status against known valid values
	validStatuses := map[string]bool{
		"processing":       true,
		"review_required":  true,
		"published":        true,
	}
	
	if validStatuses[status] {
		rqb.AddWhereCondition("status", status)
	}
	return rqb
}

// WithPagination adds pagination
func (rqb *RecipesQueryBuilder) WithPagination(limit, offset int) *RecipesQueryBuilder {
	rqb.AddOrderBy("created_at", "DESC")
	rqb.AddLimitOffset(limit, offset)
	return rqb
}