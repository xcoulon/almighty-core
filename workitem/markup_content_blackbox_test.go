package workitem_test

import (
	"testing"

	"github.com/almighty/almighty-core/markup"
	"github.com/stretchr/testify/assert"
)

func TestNewMarkupContentFromMapWithValidMarkup(t *testing.T) {
	// given
	input := make(map[string]interface{})
	input[markup.ContentKey] = "foo"
	input[markup.MarkupKey] = markup.SystemMarkupMarkdown
	// when
	result := markup.NewMarkupContentFromMap(input)
	// then
	assert.Equal(t, input[markup.ContentKey].(string), result.Content)
	assert.Equal(t, input[markup.MarkupKey].(string), result.Markup)
}

func TestNewMarkupContentFromMapWithInvalidMarkup(t *testing.T) {
	// given
	input := make(map[string]interface{})
	input[markup.ContentKey] = "foo"
	input[markup.MarkupKey] = "bar"
	// when
	result := markup.NewMarkupContentFromMap(input)
	// then
	assert.Equal(t, input[markup.ContentKey].(string), result.Content)
	assert.Equal(t, markup.SystemMarkupDefault, result.Markup)
}

func TestNewMarkupContentFromMapWithMissingMarkup(t *testing.T) {
	// given
	input := make(map[string]interface{})
	input[markup.ContentKey] = "foo"
	// when
	result := markup.NewMarkupContentFromMap(input)
	// then
	assert.Equal(t, input[markup.ContentKey].(string), result.Content)
	assert.Equal(t, markup.SystemMarkupDefault, result.Markup)
}

func TestNewMarkupContentFromMapWithEmptyMarkup(t *testing.T) {
	// given
	input := make(map[string]interface{})
	input[markup.ContentKey] = "foo"
	input[markup.MarkupKey] = ""
	// when
	result := markup.NewMarkupContentFromMap(input)
	// then
	assert.Equal(t, input[markup.ContentKey].(string), result.Content)
	assert.Equal(t, markup.SystemMarkupDefault, result.Markup)
}
