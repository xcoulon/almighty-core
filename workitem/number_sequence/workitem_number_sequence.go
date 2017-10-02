package numbersequence

import (
	"fmt"

	uuid "github.com/satori/go.uuid"
)

// WorkItemNumberSequence the sequence for work item numbers in a space
type WorkItemNumberSequence struct {
	ID         uuid.UUID `sql:"type:uuid" gorm:"primary_key"`
	SpaceID    uuid.UUID `sql:"type:uuid"`
	CurrentVal int
}

const (
	workitemNumberTableName = "work_item_number_sequences"
)

// TableName implements gorm.tabler
func (w WorkItemNumberSequence) TableName() string {
	return workitemNumberTableName
}

func (w *WorkItemNumberSequence) String() string {
	return fmt.Sprintf("SpaceId=%s Number=%d", w.SpaceID.String(), w.CurrentVal)
}