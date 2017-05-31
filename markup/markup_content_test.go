package markup

import "testing"
import "github.com/stretchr/testify/assert"

func TestGetDefaultMarkupFromNil(t *testing.T) {
	// when
	result := NilSafeGetMarkup(nil)
	// then
	assert.Equal(t, SystemMarkupDefault, result)
}

func TestGetMarkupFromValue(t *testing.T) {
	// given
	markup := SystemMarkupMarkdown
	// when
	result := NilSafeGetMarkup(&markup)
	// then
	assert.Equal(t, markup, result)
}

func TestGetMarkupFromEmptyValue(t *testing.T) {
	// given
	markup := ""
	// when
	result := NilSafeGetMarkup(&markup)
	// then
	assert.Equal(t, SystemMarkupDefault, result)
}

func TestIsMarkupSupported(t *testing.T) {
	assert.True(t, IsMarkupSupported(SystemMarkupDefault))
	assert.True(t, IsMarkupSupported(SystemMarkupPlainText))
	assert.True(t, IsMarkupSupported(SystemMarkupMarkdown))
	assert.False(t, IsMarkupSupported(""))
	assert.False(t, IsMarkupSupported("foo"))
}
