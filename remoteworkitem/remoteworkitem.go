package remoteworkitem

import (
	"encoding/json"

	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/workitem"
	"github.com/pkg/errors"
)

// List of supported attributes
const (
	ProviderGithub = "github"
	ProviderJira   = "jira"

	// The keys in the flattened response JSON of a typical Github issue.
	GithubTitle         = "title"
	GithubDescription   = "body"
	GithubState         = "state"
	GithubID            = "url"
	GithubCreator       = "user.login"
	GithubAssigneeLogin = "assignee.login"
	GithubAssigneeURL   = "assignee.url"

	// The keys in the flattened response JSON of a typical Jira issue.
	JiraTitle         = "fields.summary"
	JiraBody          = "fields.description"
	JiraState         = "fields.status.name"
	JiraID            = "self"
	JiraCreator       = "fields.creator.key"
	JiraAssigneeLogin = "fields.assignee.key"
	JiraAssigneeURL   = "fields.assignee.self"
)

// RemoteWorkItem a temporary structure that holds the relevant field values retrieved from a remote work item
type RemoteWorkItem struct {
	// The field values, according to the field type
	Fields map[string]interface{}
	// unique id per installation
	ID string
	// Name of the type of this work item
	Type string
}

const (
	remoteTitle               = workitem.SystemTitle
	remoteDescription         = workitem.SystemDescription
	remoteState               = workitem.SystemState
	remoteItemID              = workitem.SystemRemoteItemID
	remoteCreator             = workitem.SystemCreator
	remoteAssigneeLogins      = "system.assignee.login"
	remoteAssigneeProfileURLs = "system.assignee.profile_url"
)

// RemoteWorkItemKeyMaps relate remote attribute keys to internal representation
var RemoteWorkItemKeyMaps = map[string]RemoteWorkItemMap{
	ProviderGithub: {
		AttributeMapper{AttributeExpression(GithubTitle), StringConverter{}}:                                             remoteTitle,
		AttributeMapper{AttributeExpression(GithubDescription), MarkupConverter{markup: rendering.SystemMarkupMarkdown}}: remoteDescription,
		AttributeMapper{AttributeExpression(GithubState), GithubStateConverter{}}:                                        remoteState,
		AttributeMapper{AttributeExpression(GithubID), StringConverter{}}:                                                remoteItemID,
		AttributeMapper{AttributeExpression(GithubCreator), StringConverter{}}:                                           remoteCreator,
		AttributeMapper{AttributeExpression(GithubAssigneeLogin), StringConverter{}}:                                     remoteAssigneeLogins,
		AttributeMapper{AttributeExpression(GithubAssigneeURL), StringConverter{}}:                                       remoteAssigneeProfileURLs,
	},
	ProviderJira: {
		AttributeMapper{AttributeExpression(JiraTitle), StringConverter{}}:                                      remoteTitle,
		AttributeMapper{AttributeExpression(JiraBody), MarkupConverter{markup: rendering.SystemMarkupJiraWiki}}: remoteDescription,
		AttributeMapper{AttributeExpression(JiraState), JiraStateConverter{}}:                                   remoteState,
		AttributeMapper{AttributeExpression(JiraID), StringConverter{}}:                                         remoteItemID,
		AttributeMapper{AttributeExpression(JiraCreator), StringConverter{}}:                                    remoteCreator,
		AttributeMapper{AttributeExpression(JiraAssigneeLogin), StringConverter{}}:                              remoteAssigneeLogins,
		AttributeMapper{AttributeExpression(JiraAssigneeURL), StringConverter{}}:                                remoteAssigneeProfileURLs,
	},
}

type AttributeConverter interface {
	Convert(interface{}, AttributeAccessor) (interface{}, error)
}

type StateConverter interface{}

type StringConverter struct{}

// MarkupConverter converts to a 'MarkupContent' element with the given 'Markup' value
type MarkupConverter struct {
	markup string
}

type ListStringConverter struct{}

type GithubStateConverter struct{}

type JiraStateConverter struct{}

// Convert method map the external tracker item to ALM WorkItem
func (sc StringConverter) Convert(value interface{}, item AttributeAccessor) (interface{}, error) {
	return value, nil
}

// Convert returns the given `value` if the `item` is not nil`, otherwise returns `nil`
func (converter MarkupConverter) Convert(value interface{}, item AttributeAccessor) (interface{}, error) {
	// return a 'nil' result if the supplied 'value' was nil
	if value == nil {
		return nil, nil
	}
	switch value.(type) {
	case string:
		return rendering.NewMarkupContent(value.(string), converter.markup), nil
	default:
		return nil, errors.Errorf("Unexpected type of value to convert: %T", value)
	}
}

// Convert method map the external tracker item to ALM WorkItem
func (sc ListStringConverter) Convert(value interface{}, item AttributeAccessor) (interface{}, error) {
	return []interface{}{value}, nil
}

func (ghc GithubStateConverter) Convert(value interface{}, item AttributeAccessor) (interface{}, error) {
	if value.(string) == "closed" {
		value = "closed"
	} else if value.(string) == "open" {
		value = "open"
	}
	return value, nil
}

func (jhc JiraStateConverter) Convert(value interface{}, item AttributeAccessor) (interface{}, error) {
	if value.(string) == "closed" {
		value = "closed"
	} else if value.(string) == "open" {
		value = "open"
	} else if value.(string) == "in progress" {
		value = "in progress"
	} else if value.(string) == "resolved" {
		value = "resolved"
	}
	return value, nil
}

type AttributeMapper struct {
	expression         AttributeExpression
	attributeConverter AttributeConverter
}

// RemoteWorkItemMap will define mappings between remote<->internal attribute
type RemoteWorkItemMap map[AttributeMapper]string

// AttributeExpression represents a commonly understood String format for a target path
type AttributeExpression string

// AttributeAccessor defines the interface between a RemoteWorkItem and the Mapper
type AttributeAccessor interface {
	// Get returns the value based on a commonly understood attribute expression
	Get(field AttributeExpression) interface{}
}

// RemoteWorkItemImplRegistry contains all possible providers
var RemoteWorkItemImplRegistry = map[string]func(TrackerItem) (AttributeAccessor, error){
	ProviderGithub: NewGitHubRemoteWorkItem,
	ProviderJira:   NewJiraRemoteWorkItem,
}

// GitHubRemoteWorkItem knows how to implement a FieldAccessor on a GitHub Issue JSON struct
// and it should also know how to convert a value in remote work item for use in local WI
type GitHubRemoteWorkItem struct {
	issue map[string]interface{}
}

// NewGitHubRemoteWorkItem creates a new Decoded AttributeAccessor for a GitHub Issue
func NewGitHubRemoteWorkItem(item TrackerItem) (AttributeAccessor, error) {
	var j map[string]interface{}
	err := json.Unmarshal([]byte(item.Item), &j)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	j = Flatten(j)
	return GitHubRemoteWorkItem{issue: j}, nil
}

// Get attribute from issue map
func (gh GitHubRemoteWorkItem) Get(field AttributeExpression) interface{} {
	return gh.issue[string(field)]
}

// JiraRemoteWorkItem knows how to implement a FieldAccessor on a Jira Issue JSON struct
type JiraRemoteWorkItem struct {
	issue map[string]interface{}
}

// NewJiraRemoteWorkItem creates a new Decoded AttributeAccessor for a GitHub Issue
func NewJiraRemoteWorkItem(item TrackerItem) (AttributeAccessor, error) {
	var j map[string]interface{}
	err := json.Unmarshal([]byte(item.Item), &j)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	j = Flatten(j)
	return JiraRemoteWorkItem{issue: j}, nil
}

// Get attribute from issue map
func (jira JiraRemoteWorkItem) Get(field AttributeExpression) interface{} {
	return jira.issue[string(field)]
}

// Map maps the remote WorkItem to a local RemoteWorkItem
func Map(remoteItem AttributeAccessor, mapping RemoteWorkItemMap) (RemoteWorkItem, error) {
	remoteWorkItem := RemoteWorkItem{Fields: make(map[string]interface{})}
	for from, to := range mapping {
		originalValue := remoteItem.Get(from.expression)
		convertedValue, err := from.attributeConverter.Convert(originalValue, remoteItem)
		if err == nil {
			remoteWorkItem.Fields[to] = convertedValue
		}
	}
	return remoteWorkItem, nil
}
