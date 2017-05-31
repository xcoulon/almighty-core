package rendering

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/markup"
	"github.com/almighty/almighty-core/workitem"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const (
	// markdownLinkTemplate the markdown format for links:
	// [an example](http://example.com/ "Title")
	markdownLinkTemplate string = "[%[1]s](%[2]s \"%[3]s\")"
)

// MarkupRenderer the backing structure for the markup renderer service
type MarkupRenderer struct {
	workitemRepository workitem.WorkItemRepository
	baseAPI            string
}

// NewMarkupRenderer creates a markup renderer.
func NewMarkupRenderer(workitemRepository workitem.WorkItemRepository, baseAPI string) MarkupRenderer {
	return MarkupRenderer{
		workitemRepository: workitemRepository,
		baseAPI:            baseAPI,
	}
}

// RenderMarkupToHTML converts the given `content` in HTML using the markup tool corresponding to the given `markup` argument
// or return nil if no tool for the given `markup` is available, or returns an `error` if the command was not found or failed.
func (mr *MarkupRenderer) RenderMarkupToHTML(ctx context.Context, spaceID uuid.UUID, content string, markupType string) (*string, error) {
	switch markupType {
	case markup.SystemMarkupMarkdown:
		content, err := mr.insertMarkdownLinks(ctx, content, spaceID)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to render markdown content in HTML")
		}
		unsafe := MarkdownCommonHighlighter([]byte(*content))
		p := bluemonday.UGCPolicy()
		p.AllowAttrs("class").Matching(regexp.MustCompile("^language-[a-zA-Z0-9]+$|prettyprint")).OnElements("code")
		p.AllowAttrs("class").OnElements("span")
		p.RequireNoFollowOnLinks(false)
		html := string(p.SanitizeBytes(unsafe))
		return &html, nil
	default:
		// also applies when the markup type is 'markup.SystemMarkupPlainText'
		return &content, nil
	}
}

// insertLinks looks for a '#' character followed by a number corresponding to an existing work item in the given space
func (mr *MarkupRenderer) insertMarkdownLinks(ctx context.Context, source string, spaceID uuid.UUID) (*string, error) {
	re, err := regexp.Compile("#\\d+")
	if err != nil {
		return nil, errors.Wrap(err, "Unable to insert markdown links in content")
	}
	matchValues := re.FindAllString(source, -1)
	// retain the numbers for which a work item exists in the given space
	replacers := make([]string, len(matchValues)*2)
	for _, matchValue := range matchValues {
		wiNumberStr := strings.Replace(matchValue, "#", "", 1)
		wiNumber, err := strconv.Atoi(wiNumberStr)
		if err != nil {
			// ignore the match value here, it's probably not a number, but we should carry on with the remaining values
			continue
		}
		wi, err := mr.workitemRepository.Load(ctx, spaceID, wiNumber)
		if wi != nil {
			// there's a work item for the matching number, so let's make sure the match value is replaced with a valid markdown link
			wiURL := mr.baseAPI + app.WorkitemHref(wi.SpaceID, wi.ID)
			replacement := fmt.Sprintf(markdownLinkTemplate, matchValue, wiURL, wi.Fields[workitem.SystemTitle])
			replacers = append(replacers, matchValue, replacement)
		}
	}
	// now replace all matches with Markdown links in the source content
	r := strings.NewReplacer(replacers...)
	result := r.Replace(source)
	return &result, nil
}
