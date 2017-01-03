package rendering_test

import "github.com/almighty/almighty-core/rendering"
import "testing"

func TestRenderAsciidoctorContent(t *testing.T) {
	content := "Hello, `World`!"
	result, err := rendering.RenderAsciidocToHTML(content)

	if err != nil {
		t.Error(err)
	}
	t.Log(result)
}
