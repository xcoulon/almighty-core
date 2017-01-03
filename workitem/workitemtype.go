package workitem

import (
	"strconv"
	"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
)

// String constants for the local work item types.
const (
	// pathSep specifies the symbol used to concatenate WIT names to form a so called "path"
	pathSep = "/"

	SystemRemoteItemID      = "system.remote_item_id"
	SystemTitle             = "system.title"
	SystemDescription       = "system.description"
	SystemDescriptionMarkup = "system.description.markup"
	SystemState             = "system.state"
	SystemAssignees         = "system.assignees"
	SystemCreator           = "system.creator"
	SystemCreatedAt         = "system.created_at"
	SystemIteration         = "system.iteration"

	// base item type with common fields for planner item types like userstory, experience, bug, feature, etc.
	SystemPlannerItem = "system.planneritem"

	SystemUserStory        = "system.userstory"
	SystemValueProposition = "system.valueproposition"
	SystemFundamental      = "system.fundamental"
	SystemExperience       = "system.experience"
	SystemFeature          = "system.feature"
	SystemBug              = "system.bug"

	SystemStateOpen       = "open"
	SystemStateNew        = "new"
	SystemStateInProgress = "in progress"
	SystemStateResolved   = "resolved"
	SystemStateClosed     = "closed"

	SystemMarkupPlainText = "PlainText"
	SystemMarkupMarkdown  = "Markdown"
	SystemMarkupJiraWiki  = "JiraWiki"
	SystemMarkupAsciidoc  = "Asciidoc"
	SystemMarkupHTML      = "HTML"
)

// WorkItemType represents a work item type as it is stored in the db
type WorkItemType struct {
	gormsupport.Lifecycle
	// the unique name of this work item type.
	Name string `gorm:"primary_key"`
	// Version for optimistic concurrency control
	Version int
	// the id's of the parents, separated with some separator
	Path string
	// definitions of the fields this work item type supports
	Fields FieldDefinitions `sql:"type:jsonb"`
}

// TableName implements gorm.tabler
func (wit WorkItemType) TableName() string {
	return "work_item_types"
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemType{}
var _ convert.Equaler = (*WorkItemType)(nil)

// Equal returns true if two WorkItemType objects are equal; otherwise false is returned.
func (wit WorkItemType) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemType)
	if !ok {
		return false
	}
	if !wit.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if wit.Version != other.Version {
		return false
	}
	if wit.Name != other.Name {
		return false
	}
	if wit.Path != other.Path {
		return false
	}
	if len(wit.Fields) != len(other.Fields) {
		return false
	}
	for witKey, witVal := range wit.Fields {
		otherVal, keyFound := other.Fields[witKey]
		if !keyFound {
			return false
		}
		if !witVal.Equal(otherVal) {
			return false
		}
	}
	return true
}

// ConvertFromModel serializes a database persisted workitem.
func (wit WorkItemType) ConvertFromModel(workItem WorkItem) (*app.WorkItem, error) {
	result := app.WorkItem{
		ID:      strconv.FormatUint(workItem.ID, 10),
		Type:    workItem.Type,
		Version: workItem.Version,
		Fields:  map[string]interface{}{}}

	for name, field := range wit.Fields {
		var err error
		if name == SystemCreatedAt {
			continue
		}
		result.Fields[name], err = field.ConvertFromModel(name, workItem.Fields[name])
		if err != nil {
			return nil, err
		}
	}

	return &result, nil
}

// IsTypeOrSubtypeOf returns true if the work item type is of the given type name,
// or a subtype; otherwise false is returned.
func (wit WorkItemType) IsTypeOrSubtypeOf(typeName string) bool {
	// Remove any prefixed "/"
	for strings.HasPrefix(typeName, pathSep) && len(typeName) > 0 {
		typeName = strings.TrimPrefix(typeName, pathSep)
	}
	// Remove any trailing "/"
	for strings.HasSuffix(typeName, pathSep) && len(typeName) > 0 {
		typeName = strings.TrimSuffix(typeName, pathSep)
	}
	if len(typeName) <= 0 {
		return false
	}
	// Check for complete inclusion (e.g. "/bar/" is contained in "/foo/bar/cake")
	// and for suffix (e.g. "/cake" is the suffix of "/foo/bar/cake").
	return wit.Name == typeName || strings.Contains(wit.Path, pathSep+typeName+pathSep)
}
