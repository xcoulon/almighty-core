package search

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"testing"

	c "github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	t.Run(Q_EQ, func(t *testing.T) {
		t.Parallel()
		// given
		input := fmt.Sprintf(`{"space": { "%s": "openshiftio"}}`, Q_EQ)
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.Nil(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		openshiftio := "openshiftio"
		expectedQuery := Query{Name: "space", Value: &openshiftio}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run(Q_NE, func(t *testing.T) {
		t.Parallel()
		// given
		input := fmt.Sprintf(`{"space": { "%s": "openshiftio"}}`, Q_NE)
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.Nil(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		openshiftio := "openshiftio"
		expectedQuery := Query{Name: "space", Value: &openshiftio, Negate: true}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	// {"type" : { "$IN" : ["", "" , ""] } }
	t.Run(Q_AND, func(t *testing.T) {
		t.Parallel()
		// given
		input := `{"` + Q_AND + `": [{"space": "openshiftio"}, {"status": "NEW"}]}`
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.Nil(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		openshiftio := "openshiftio"
		status := "NEW"
		expectedQuery := Query{Name: Q_AND, Children: []Query{
			{Name: "space", Value: &openshiftio},
			{Name: "status", Value: &status}},
		}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run("AND with EQ", func(t *testing.T) {
		t.Parallel()
		// given
		input := `{"` + Q_AND + `": [{"space": {"$EQ": "openshiftio"}}, {"status": "NEW"}]}`
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.Nil(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		openshiftio := "openshiftio"
		status := "NEW"
		expectedQuery := Query{Name: Q_AND, Children: []Query{
			{Name: "space", Value: &openshiftio},
			{Name: "status", Value: &status}},
		}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run("Minimal OR and AND operation", func(t *testing.T) {
		t.Parallel()
		input := `
			{"` + Q_OR + `": [{"` + Q_AND + `": [{"space": "openshiftio"},
                         {"area": "planner"}]},
	        {"` + Q_AND + `": [{"space": "rhel"}]}]}`
		fm := map[string]interface{}{}

		// Parsing/Unmarshalling JSON encoding/json
		err := json.Unmarshal([]byte(input), &fm)
		require.Nil(t, err)
		q := &Query{}

		parseMap(fm, q)

		openshiftio := "openshiftio"
		area := "planner"
		rhel := "rhel"
		expected := &Query{Name: Q_OR, Children: []Query{
			{Name: Q_AND, Children: []Query{
				{Name: "space", Value: &openshiftio},
				{Name: "area", Value: &area}}},
			{Name: Q_AND, Children: []Query{
				{Name: "space", Value: &rhel}}},
		}}
		assert.Equal(t, expected, q)
	})

	t.Run("minimal OR and AND and Negate operation", func(t *testing.T) {
		t.Parallel()
		input := `
		{"` + Q_OR + `": [{"` + Q_AND + `": [{"space": "openshiftio"},
                         {"area": "planner"}]},
			 {"` + Q_AND + `": [{"space": "rhel", "negate": true}]}]}`
		fm := map[string]interface{}{}

		// Parsing/Unmarshalling JSON encoding/json
		err := json.Unmarshal([]byte(input), &fm)
		require.Nil(t, err)
		q := &Query{}

		parseMap(fm, q)

		openshiftio := "openshiftio"
		area := "planner"
		rhel := "rhel"
		expected := &Query{Name: Q_OR, Children: []Query{
			{Name: Q_AND, Children: []Query{
				{Name: "space", Value: &openshiftio},
				{Name: "area", Value: &area}}},
			{Name: Q_AND, Children: []Query{
				{Name: "space", Value: &rhel, Negate: true}}},
		}}
		assert.Equal(t, expected, q)
	})

	t.Run("minimal OR and AND and Negate operation with EQ", func(t *testing.T) {
		t.Parallel()
		input := `
		{"` + Q_OR + `": [{"` + Q_AND + `": [{"space": "openshiftio"},
                         {"area": "planner"}]},
			 {"` + Q_AND + `": [{"space": {"$EQ": "rhel"}, "negate": true}]}]}`
		fm := map[string]interface{}{}

		// Parsing/Unmarshalling JSON encoding/json
		err := json.Unmarshal([]byte(input), &fm)
		require.Nil(t, err)
		q := &Query{}

		parseMap(fm, q)

		openshiftio := "openshiftio"
		area := "planner"
		rhel := "rhel"
		expected := &Query{Name: Q_OR, Children: []Query{
			{Name: Q_AND, Children: []Query{
				{Name: "space", Value: &openshiftio},
				{Name: "area", Value: &area}}},
			{Name: Q_AND, Children: []Query{
				{Name: "space", Value: &rhel, Negate: true}}},
		}}
		assert.Equal(t, expected, q)
	})

	t.Run(Q_IN, func(t *testing.T) {
		t.Parallel()
		// given
		input := fmt.Sprintf(`{"status": { "%s": ["NEW", "OPEN"]}}`, Q_IN)
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.Nil(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		new := "NEW"
		open := "OPEN"
		expectedQuery := Query{Name: Q_OR, Children: []Query{
			{Name: "status", Value: &new},
			{Name: "status", Value: &open}},
		}
		assert.Equal(t, expectedQuery, actualQuery)
	})

}
func TestGenerateExpression(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	t.Run("Equals (top-level)", func(t *testing.T) {
		t.Parallel()
		// given
		spaceName := "openshiftio"
		q := Query{Name: "space", Value: &spaceName}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.Equals(
			c.Field("SpaceID"),
			c.Literal(spaceName),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run(Q_NOT+" (top-level)", func(t *testing.T) {
		t.Parallel()
		// given
		spaceName := "openshiftio"
		q := Query{Name: "space", Value: &spaceName, Negate: true}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.Not(
			c.Field("SpaceID"),
			c.Literal(spaceName),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})
	t.Run(Q_AND, func(t *testing.T) {
		t.Parallel()
		// given
		statusName := "NEW"
		spaceName := "openshiftio"
		q := Query{
			Name: Q_AND,
			Children: []Query{
				{Name: "space", Value: &spaceName},
				{Name: "state", Value: &statusName},
			},
		}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.And(
			c.Equals(
				c.Field("SpaceID"),
				c.Literal(spaceName),
			),
			c.Equals(
				c.Field("system.state"),
				c.Literal(statusName),
			),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run(Q_OR, func(t *testing.T) {
		t.Parallel()
		// given
		statusName := "NEW"
		spaceName := "openshiftio"
		q := Query{
			Name: Q_OR,
			Children: []Query{
				{Name: "space", Value: &spaceName},
				{Name: "state", Value: &statusName},
			},
		}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.Or(
			c.Equals(
				c.Field("SpaceID"),
				c.Literal(spaceName),
			),
			c.Equals(
				c.Field("system.state"),
				c.Literal(statusName),
			),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run(Q_NOT+" (nested)", func(t *testing.T) {
		t.Parallel()
		// given
		statusName := "NEW"
		spaceName := "openshiftio"
		q := Query{
			Name: Q_AND,
			Children: []Query{
				{Name: "space", Value: &spaceName, Negate: true},
				{Name: "state", Value: &statusName},
			},
		}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.And(
			c.Not(
				c.Field("SpaceID"),
				c.Literal(spaceName),
			),
			c.Equals(
				c.Field("system.state"),
				c.Literal(statusName),
			),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run("NULL value", func(t *testing.T) {
		t.Parallel()
		// given
		spaceName := "openshiftio"
		q := Query{
			Name: Q_AND,
			Children: []Query{
				{Name: "space", Value: &spaceName},
				{Name: "assignee", Value: nil},
			},
		}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.And(
			c.Equals(
				c.Field("SpaceID"),
				c.Literal(spaceName),
			),

			c.IsNull("system.assignees"),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run("NULL value at top-level", func(t *testing.T) {
		t.Parallel()
		// given
		q := Query{
			Name: "assignee", Value: nil,
		}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.IsNull("system.assignees")

		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run("NULL value at top-level with Negate", func(t *testing.T) {
		t.Parallel()
		// given
		q := Query{
			Name: "assignee", Value: nil, Negate: true,
		}
		// when
		actualExpr, err := q.generateExpression()
		// then
		require.NotNil(t, err)
		require.Nil(t, actualExpr)
		assert.Contains(t, err.Error(), "negate for null not supported")
	})

	t.Run("NULL value with Negate", func(t *testing.T) {
		t.Parallel()
		// given
		spaceName := "openshiftio"
		q := Query{
			Name: Q_AND,
			Children: []Query{
				{Name: "space", Value: &spaceName},
				{Name: "assignee", Value: nil, Negate: true},
			},
		}
		// when
		actualExpr, err := q.generateExpression()
		// then
		require.NotNil(t, err)
		require.Nil(t, actualExpr)
		assert.Contains(t, err.Error(), "negate for null not supported")
	})

}

func expectEqualExpr(t *testing.T, expectedExpr, actualExpr c.Expression) {
	actualClause, actualParameters, actualErrs := workitem.Compile(actualExpr)
	if len(actualErrs) > 0 {
		debug.PrintStack()
		require.Nil(t, actualErrs, "failed to compile actual expression")
	}
	exprectedClause, expectedParameters, expectedErrs := workitem.Compile(expectedExpr)
	if len(expectedErrs) > 0 {
		debug.PrintStack()
		require.Nil(t, expectedErrs, "failed to compile expected expression")
	}
	require.Equal(t, exprectedClause, actualClause, "where clause differs")
	require.Equal(t, expectedParameters, actualParameters, "parameters differ")
}

func TestGenerateExpressionWithNonExistingKey(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	t.Run("Empty query", func(t *testing.T) {
		t.Parallel()
		// given
		q := Query{}
		// when
		actualExpr, err := q.generateExpression()
		// then
		require.NotNil(t, err)
		require.Nil(t, actualExpr)
	})
	t.Run("Empty name", func(t *testing.T) {
		t.Parallel()
		// given
		spaceName := "openshiftio"
		q := Query{Name: "", Value: &spaceName}
		// when
		actualExpr, err := q.generateExpression()
		// then
		require.NotNil(t, err)
		require.Nil(t, actualExpr)
	})

	t.Run("No existing key", func(t *testing.T) {
		t.Parallel()
		// given
		spaceName := "openshiftio"
		q := Query{Name: "nonexistingkey", Value: &spaceName}
		// when
		actualExpr, err := q.generateExpression()
		// then
		require.NotNil(t, err)
		require.Nil(t, actualExpr)
	})

}
