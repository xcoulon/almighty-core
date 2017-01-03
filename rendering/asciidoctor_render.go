package rendering

import (
	"bytes"
	"os/exec"
)

// RenderAsciidocToHTML converts the given `content` in HTML using the `asciidoctor` command
// that should be available, or returns an `error` if the command was not found or failed.
// Note: code is based on https://github.com/spf13/hugo/pull/826/files (ASL 2.0 license)
func RenderAsciidocToHTML(content string) (string, error) {
	path, err := exec.LookPath("asciidoctor")
	if err != nil {
		return "", err
	}
	cmd := exec.Command(path, "--safe", "--no-header-footer", "-")
	cmd.Stdin = bytes.NewReader([]byte(content))
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}
