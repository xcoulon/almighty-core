package workitem

import (
	"strconv"

	"golang.org/x/net/context"

	"fmt"

	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/errors"

	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/markup"

	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const orderValue = 1000

type DirectionType string

const (
	DirectionAbove  DirectionType = "above"
	DirectionBelow  DirectionType = "below"
	DirectionTop    DirectionType = "top"
	DirectionBottom DirectionType = "bottom"
)

// WorkItemRepository encapsulates storage & retrieval of work items
type WorkItemRepository interface {
	LoadByID(ctx context.Context, id uuid.UUID) (*WorkItem, error)
	Load(ctx context.Context, spaceID uuid.UUID, wiNumber int) (*WorkItem, error)
	Save(ctx context.Context, spaceID uuid.UUID, wi WorkItem, modifierID uuid.UUID) (*WorkItem, error)
	Reorder(ctx context.Context, direction DirectionType, targetID *uuid.UUID, wi WorkItem, modifierID uuid.UUID) (*WorkItem, error)
	Delete(ctx context.Context, id uuid.UUID, suppressorID uuid.UUID) error
	Create(ctx context.Context, spaceID uuid.UUID, typeID uuid.UUID, fields map[string]interface{}, creatorID uuid.UUID) (*WorkItem, error)
	List(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression, parentExists *bool, start *int, length *int) ([]WorkItem, int, error)
	Fetch(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression) (*WorkItem, error)
	GetCountsPerIteration(ctx context.Context, spaceID uuid.UUID) (map[string]WICountsPerIteration, error)
	GetCountsForIteration(ctx context.Context, iterationID uuid.UUID) (map[string]WICountsPerIteration, error)
	Count(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression) (int, error)
}

// NewWorkItemRepository creates a GormWorkItemRepository
func NewWorkItemRepository(db *gorm.DB) *GormWorkItemRepository {
	repository := &GormWorkItemRepository{db, &GormWorkItemTypeRepository{db}, &GormRevisionRepository{db}}
	return repository
}

// GormWorkItemRepository implements WorkItemRepository using gorm
type GormWorkItemRepository struct {
	db   *gorm.DB
	witr *GormWorkItemTypeRepository
	wirr *GormRevisionRepository
}

// ************************************************
// WorkItemRepository implementation
// ************************************************

// LoadFromDB returns the work item with the given natural ID in model representation.
func (r *GormWorkItemRepository) LoadFromDB(ctx context.Context, id uuid.UUID) (*WorkItemStorage, error) {
	log.Info(nil, map[string]interface{}{
		"wi_id": id,
	}, "Loading work item")

	res := WorkItemStorage{}
	tx := r.db.Model(WorkItemStorage{}).Where("id = ?", id).First(&res)
	if tx.RecordNotFound() {
		log.Error(nil, map[string]interface{}{
			"wi_id": id,
		}, "work item not found")
		return nil, errors.NewNotFoundError("work item", id.String())
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(tx.Error.Error())
	}
	return &res, nil
}

// LoadByID returns the work item for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) LoadByID(ctx context.Context, id uuid.UUID) (*WorkItem, error) {
	res, err := r.LoadFromDB(ctx, id)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	wiType, err := r.witr.LoadTypeFromDB(ctx, res.Type)
	if err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	return ConvertWorkItemStorageToModel(wiType, res)
}

// Load returns the work item for the given spaceID and item id
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) Load(ctx context.Context, spaceID uuid.UUID, wiNumber int) (*WorkItem, error) {
	wiStorage, wiType, err := r.loadWorkItemStorage(ctx, spaceID, wiNumber, false)
	if err != nil {
		return nil, err
	}
	return ConvertWorkItemStorageToModel(wiType, wiStorage)
}

func (r *GormWorkItemRepository) loadWorkItemStorage(ctx context.Context, spaceID uuid.UUID, wiNumber int, selectForUpdate bool) (*WorkItemStorage, *WorkItemType, error) {
	log.Info(nil, map[string]interface{}{
		"wi_number": wiNumber,
		"space_id":  spaceID,
	}, "Loading work item")
	wiStorage := &WorkItemStorage{}
	// SELECT ... FOR UPDATE will lock the row to prevent concurrent update while until surrounding transaction ends.
	tx := r.db
	if selectForUpdate {
		tx = tx.Set("gorm:query_option", "FOR UPDATE")
	}
	tx = tx.Model(wiStorage).Where("number=? AND space_id=?", wiNumber, spaceID).First(wiStorage)
	if tx.RecordNotFound() {
		log.Error(nil, map[string]interface{}{
			"wi_number": wiNumber,
			"space_id":  spaceID,
		}, "work item not found")
		return nil, nil, errors.NewNotFoundError("work item", strconv.Itoa(wiNumber))
	}
	if tx.Error != nil {
		return nil, nil, errors.NewInternalError(tx.Error.Error())
	}
	wiType, err := r.witr.LoadTypeFromDB(ctx, wiStorage.Type)
	if err != nil {
		return nil, nil, errors.NewInternalError(err.Error())
	}
	return wiStorage, wiType, nil
}

// LoadTopWorkitem returns top most work item of the list. Top most workitem has the Highest order.
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) LoadTopWorkitem(ctx context.Context) (*WorkItem, error) {
	res := WorkItemStorage{}
	db := r.db.Model(WorkItemStorage{})
	query := fmt.Sprintf("execution_order = (SELECT max(execution_order) FROM %[1]s)",
		WorkItemStorage{}.TableName(),
	)
	db = db.Where(query).First(&res)
	wiType, err := r.witr.LoadTypeFromDB(ctx, res.Type)
	if err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	return ConvertWorkItemStorageToModel(wiType, &res)
}

// LoadBottomWorkitem returns bottom work item of the list. Bottom most workitem has the lowest order.
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) LoadBottomWorkitem(ctx context.Context) (*WorkItem, error) {
	res := WorkItemStorage{}
	db := r.db.Model(WorkItemStorage{})
	query := fmt.Sprintf("execution_order = (SELECT min(execution_order) FROM %[1]s)",
		WorkItemStorage{}.TableName(),
	)
	db = db.Where(query).First(&res)
	wiType, err := r.witr.LoadTypeFromDB(ctx, res.Type)
	if err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	return ConvertWorkItemStorageToModel(wiType, &res)
}

// LoadHighestOrder returns the highest order
func (r *GormWorkItemRepository) LoadHighestOrder() (float64, error) {
	res := WorkItemStorage{}
	db := r.db.Model(WorkItemStorage{})
	query := fmt.Sprintf("execution_order = (SELECT max(execution_order) FROM %[1]s)",
		WorkItemStorage{}.TableName(),
	)
	db = db.Where(query).First(&res)
	order, err := strconv.ParseFloat(fmt.Sprintf("%v", res.ExecutionOrder), 64)
	if err != nil {
		return 0, errors.NewInternalError(err.Error())
	}
	return order, nil
}

// Delete deletes the work item with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemRepository) Delete(ctx context.Context, workitemID uuid.UUID, suppressorID uuid.UUID) error {
	var workItem = WorkItemStorage{}
	workItem.ID = workitemID
	// retrieve the current version of the work item to delete
	r.db.Select("id, version, type").Where("id = ?", workitemID).Find(&workItem)
	// delete the work item
	tx := r.db.Delete(workItem)
	if err := tx.Error; err != nil {
		return errors.NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.NewNotFoundError("work item", workitemID.String())
	}
	// store a revision of the deleted work item
	if err := r.wirr.Create(context.Background(), suppressorID, RevisionTypeDelete, workItem); err != nil {
		return errs.Wrapf(err, "error while deleting work item")
	}
	log.Debug(ctx, map[string]interface{}{"wi_id": workitemID}, "Work item deleted successfully!")
	return nil
}

// CalculateOrder calculates the order of the reorder workitem
func (r *GormWorkItemRepository) CalculateOrder(above, below *float64) float64 {
	return (*above + *below) / 2
}

// FindSecondItem returns the order of the second workitem required to reorder.
// Reordering a workitem requires order of two closest workitems: above and below.
// If direction == "above", then
//	FindFirstItem returns the value above which reorder item has to be placed
//      FindSecondItem returns the value below which reorder item has to be placed
// If direction == "below", then
//	FindFirstItem returns the value below which reorder item has to be placed
//      FindSecondItem returns the value above which reorder item has to be placed
func (r *GormWorkItemRepository) FindSecondItem(order *float64, secondItemDirection DirectionType) (*uuid.UUID, *float64, error) {
	Item := WorkItemStorage{}
	var tx *gorm.DB
	switch secondItemDirection {
	case DirectionAbove:
		// Finds the item above which reorder item has to be placed
		tx = r.db.Where(fmt.Sprintf("execution_order = (SELECT max(execution_order) FROM %s WHERE (execution_order < ?))", WorkItemStorage{}.TableName()), order).First(&Item)
	case DirectionBelow:
		// Finds the item below which reorder item has to be placed
		tx = r.db.Where(fmt.Sprintf("execution_order = (SELECT min(execution_order) FROM %s WHERE (execution_order > ?))", WorkItemStorage{}.TableName()), order).First(&Item)
	default:
		return nil, nil, nil
	}
	if tx.RecordNotFound() {
		// Item is placed at first or last position
		ItemID := Item.ID
		return &ItemID, nil, nil
	}
	if tx.Error != nil {
		return nil, nil, errors.NewInternalError(tx.Error.Error())
	}
	ItemID := Item.ID
	return &ItemID, &Item.ExecutionOrder, nil
}

// FindFirstItem returns the order of the target workitem
func (r *GormWorkItemRepository) FindFirstItem(id uuid.UUID) (*float64, error) {
	res := WorkItemStorage{}
	tx := r.db.Model(WorkItemStorage{}).Where("id = ?", id).First(&res)
	if tx.RecordNotFound() {
		return nil, errors.NewNotFoundError("work item", id.String())
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(tx.Error.Error())
	}
	return &res.ExecutionOrder, nil
}

// Reorder places the to-be-reordered workitem above the input workitem.
// The order of workitems are spaced by a factor of 1000.
// The new order of workitem := (order of previousitem + order of nextitem)/2
// Version must be the same as the one int the stored version
func (r *GormWorkItemRepository) Reorder(ctx context.Context, direction DirectionType, targetID *uuid.UUID, wi WorkItem, modifierID uuid.UUID) (*WorkItem, error) {
	var order float64
	res := WorkItemStorage{}
	tx := r.db.Model(WorkItemStorage{}).Where("id = ?", wi.ID).First(&res)
	if tx.RecordNotFound() {
		return nil, errors.NewNotFoundError("work item", wi.ID.String())
	}
	if err := tx.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	if res.Version != wi.Version {
		log.Info(ctx, map[string]interface{}{
			"wi_id":           wi.ID.String(),
			"current_version": res.Version,
			"input_version":   wi.Version},
			"version_conflict while reordering items")
		return nil, errors.NewVersionConflictError("version conflict")
	}

	wiType, err := r.witr.LoadTypeFromDB(ctx, wi.Type)
	if err != nil {
		return nil, errors.NewBadParameterError("Type", wi.Type)
	}

	switch direction {
	case DirectionBelow:
		// if direction == "below", place the reorder item **below** the workitem having id equal to targetID
		aboveItemOrder, err := r.FindFirstItem(*targetID)
		if aboveItemOrder == nil || err != nil {
			return nil, errors.NewNotFoundError("work item", targetID.String())
		}
		belowItemID, belowItemOrder, err := r.FindSecondItem(aboveItemOrder, DirectionAbove)
		if err != nil {
			return nil, errors.NewNotFoundError("work item", targetID.String())
		}
		if belowItemOrder == nil {
			// Item is placed at last position
			belowItemOrder := float64(0)
			order = r.CalculateOrder(aboveItemOrder, &belowItemOrder)
		} else if *belowItemID == res.ID {
			// When same reorder request is made again
			order = wi.Fields[SystemOrder].(float64)
		} else {
			order = r.CalculateOrder(aboveItemOrder, belowItemOrder)
		}
	case DirectionAbove:
		// if direction == "above", place the reorder item **above** the workitem having id equal to targetID
		belowItemOrder, _ := r.FindFirstItem(*targetID)
		if belowItemOrder == nil || err != nil {
			return nil, errors.NewNotFoundError("work item", targetID.String())
		}
		aboveItemID, aboveItemOrder, err := r.FindSecondItem(belowItemOrder, DirectionBelow)
		if err != nil {
			return nil, errors.NewNotFoundError("work item", targetID.String())
		}
		if aboveItemOrder == nil {
			// Item is placed at first position
			order = *belowItemOrder + float64(orderValue)
		} else if *aboveItemID == res.ID {
			// When same reorder request is made again
			order = wi.Fields[SystemOrder].(float64)
		} else {
			order = r.CalculateOrder(aboveItemOrder, belowItemOrder)
		}
	case DirectionTop:
		// if direction == "top", place the reorder item at the topmost position. Now, the reorder item has the highest order in the whole list.
		res, err := r.LoadTopWorkitem(ctx)
		if err != nil {
			return nil, errs.Wrapf(err, "Failed to reorder")
		}
		if wi.ID == res.ID {
			// When same reorder request is made again
			order = wi.Fields[SystemOrder].(float64)
		} else {
			topItemOrder := res.Fields[SystemOrder].(float64)
			order = topItemOrder + orderValue
		}
	case DirectionBottom:
		// if direction == "bottom", place the reorder item at the bottom most position. Now, the reorder item has the lowest order in the whole list
		res, err := r.LoadBottomWorkitem(ctx)
		if err != nil {
			return nil, errs.Wrapf(err, "Failed to reorder")
		}
		if wi.ID == res.ID {
			// When same reorder request is made again
			order = wi.Fields[SystemOrder].(float64)
		} else {
			bottomItemOrder := res.Fields[SystemOrder].(float64)
			order = bottomItemOrder / 2
		}
	default:
		return &wi, nil
	}
	res.Version = res.Version + 1
	res.Type = wi.Type
	res.Fields = Fields{}

	res.ExecutionOrder = order

	for fieldName, fieldDef := range wiType.Fields {
		if fieldName == SystemCreatedAt || fieldName == SystemUpdatedAt || fieldName == SystemOrder {
			continue
		}
		fieldValue := wi.Fields[fieldName]
		var err error
		res.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return nil, errors.NewBadParameterError(fieldName, fieldValue)
		}
	}
	tx = tx.Where("Version = ?", wi.Version).Save(&res)
	if err := tx.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	// store a revision of the modified work item
	err = r.wirr.Create(context.Background(), modifierID, RevisionTypeUpdate, res)
	if err != nil {
		return nil, err
	}
	return ConvertWorkItemStorageToModel(wiType, &res)
}

// Save updates the given work item in storage. Version must be the same as the one int the stored version
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemRepository) Save(ctx context.Context, spaceID uuid.UUID, updatedWorkItem WorkItem, modifierID uuid.UUID) (*WorkItem, error) {
	wiStorage, wiType, err := r.loadWorkItemStorage(ctx, spaceID, updatedWorkItem.Number, true)
	if err != nil {
		return nil, err
	}
	if wiStorage.Version != updatedWorkItem.Version {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	wiStorage.Version = wiStorage.Version + 1
	wiStorage.Type = updatedWorkItem.Type
	wiStorage.Fields = Fields{}
	wiStorage.ExecutionOrder = updatedWorkItem.Fields[SystemOrder].(float64)
	for fieldName, fieldDef := range wiType.Fields {
		if fieldName == SystemCreatedAt || fieldName == SystemUpdatedAt || fieldName == SystemOrder {
			continue
		}
		fieldValue := updatedWorkItem.Fields[fieldName]
		var err error
		wiStorage.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return nil, errors.NewBadParameterError(fieldName, fieldValue)
		}
	}
	tx := r.db.Where("Version = ?", updatedWorkItem.Version).Save(&wiStorage)
	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"wi_id":    updatedWorkItem.ID,
			"space_id": spaceID,
			"version":  updatedWorkItem.Version,
			"err":      err,
		}, "unable to save new version of the work item")
		return nil, errors.NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	// store a revision of the modified work item
	err = r.wirr.Create(context.Background(), modifierID, RevisionTypeUpdate, *wiStorage)
	if err != nil {
		return nil, errs.Wrapf(err, "error while saving work item")
	}
	log.Info(ctx, map[string]interface{}{
		"wi_id":    updatedWorkItem.ID,
		"space_id": spaceID,
	}, "Updated work item repository")
	return ConvertWorkItemStorageToModel(wiType, wiStorage)
}

// Create creates a new work item in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemRepository) Create(ctx context.Context, spaceID uuid.UUID, typeID uuid.UUID, fields map[string]interface{}, creatorID uuid.UUID) (*WorkItem, error) {
	wiType, err := r.witr.LoadTypeFromDB(ctx, typeID)
	if err != nil {
		return nil, errors.NewBadParameterError("typeID", typeID)
	}
	// retrieve the current issue number in the given space
	numberSequence := WorkItemNumberSequence{}
	tx := r.db.Model(&WorkItemNumberSequence{}).Set("gorm:query_option", "FOR UPDATE").Where("space_id = ?", spaceID).First(&numberSequence)
	if tx.RecordNotFound() {
		numberSequence.SpaceID = spaceID
		numberSequence.CurrentVal = 1
	} else {
		numberSequence.CurrentVal++
	}
	if err = r.db.Save(&numberSequence).Error; err != nil {
		return nil, errs.Wrapf(err, "failed to create work item")
	}

	// The order of workitems are spaced by a factor of 1000.
	pos, err := r.LoadHighestOrder()
	if err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	pos = pos + orderValue
	wi := WorkItemStorage{
		Type:           typeID,
		Fields:         Fields{},
		ExecutionOrder: pos,
		SpaceID:        spaceID,
		Number:         numberSequence.CurrentVal,
	}
	fields[SystemCreator] = creatorID.String()
	for fieldName, fieldDef := range wiType.Fields {
		if fieldName == SystemCreatedAt || fieldName == SystemUpdatedAt || fieldName == SystemOrder {
			continue
		}
		fieldValue := fields[fieldName]
		var err error
		wi.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return nil, errors.NewBadParameterError(fieldName, fieldValue)
		}
		if fieldName == SystemDescription && wi.Fields[fieldName] != nil {
			description := markup.NewMarkupContentFromMap(wi.Fields[fieldName].(map[string]interface{}))
			if !markup.IsMarkupSupported(description.Markup) {
				return nil, errors.NewBadParameterError(fieldName, fieldValue)
			}
		}
	}
	if err = r.db.Create(&wi).Error; err != nil {
		return nil, errs.Wrapf(err, "failed to create work item")
	}

	witem, err := ConvertWorkItemStorageToModel(wiType, &wi)
	if err != nil {
		return nil, err
	}
	// store a revision of the created work item
	err = r.wirr.Create(context.Background(), creatorID, RevisionTypeCreate, wi)
	if err != nil {
		return nil, errs.Wrapf(err, "error while creating work item")
	}
	log.Debug(ctx, map[string]interface{}{"pkg": "workitem", "wi_id": wi.ID}, "Work item created successfully!")
	return witem, nil
}

// ConvertWorkItemStorageToModel convert work item model to app WI
func ConvertWorkItemStorageToModel(wiType *WorkItemType, wi *WorkItemStorage) (*WorkItem, error) {
	result, err := wiType.ConvertWorkItemStorageToModel(*wi)
	if err != nil {
		return nil, errors.NewConversionError(err.Error())
	}
	if _, ok := wiType.Fields[SystemCreatedAt]; ok {
		result.Fields[SystemCreatedAt] = wi.CreatedAt
	}
	if _, ok := wiType.Fields[SystemUpdatedAt]; ok {
		result.Fields[SystemUpdatedAt] = wi.UpdatedAt
	}
	if _, ok := wiType.Fields[SystemOrder]; ok {
		result.Fields[SystemOrder] = wi.ExecutionOrder
	}
	return result, nil

}

// extracted this function from List() in order to close the rows object with "defer" for more readability
// workaround for https://github.com/lib/pq/issues/81
func (r *GormWorkItemRepository) listItemsFromDB(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression, parentExists *bool, start *int, limit *int) ([]WorkItemStorage, int, error) {
	where, parameters, compileError := Compile(criteria)
	if compileError != nil {
		return nil, 0, errors.NewBadParameterError("expression", criteria)
	}
	where = where + " AND space_id = ?"
	parameters = append(parameters, spaceID.String())

	if parentExists != nil && !*parentExists {
		where += ` AND
			id not in (
				SELECT target_id FROM work_item_links
				WHERE link_type_id IN (
					SELECT id FROM work_item_link_types WHERE forward_name = 'parent of'
				)
			)`

	}
	db := r.db.Model(&WorkItemStorage{}).Where(where, parameters...)
	orgDB := db
	if start != nil {
		if *start < 0 {
			return nil, 0, errors.NewBadParameterError("start", *start)
		}
		db = db.Offset(*start)
	}
	if limit != nil {
		if *limit <= 0 {
			return nil, 0, errors.NewBadParameterError("limit", *limit)
		}
		db = db.Limit(*limit)
	}

	db = db.Select("count(*) over () as cnt2 , *").Order("execution_order desc")

	rows, err := db.Rows()
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}
	defer rows.Close()

	result := []WorkItemStorage{}
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, errors.NewInternalError(err.Error())
	}

	// need to set up a result for Scan() in order to extract total count.
	var count int
	var ignore interface{}
	columnValues := make([]interface{}, len(columns))

	for index := range columnValues {
		columnValues[index] = &ignore
	}
	columnValues[0] = &count
	first := true

	for rows.Next() {
		value := WorkItemStorage{}
		db.ScanRows(rows, &value)
		if first {
			first = false
			if err = rows.Scan(columnValues...); err != nil {
				return nil, 0, errors.NewInternalError(err.Error())
			}
		}
		result = append(result, value)

	}
	if first {
		// means 0 rows were returned from the first query (maybe becaus of offset outside of total count),
		// need to do a count(*) to find out total
		orgDB := orgDB.Select("count(*)")
		rows2, err := orgDB.Rows()
		defer rows2.Close()
		if err != nil {
			return nil, 0, errs.WithStack(err)
		}
		rows2.Next() // count(*) will always return a row
		rows2.Scan(&count)
	}
	return result, count, nil
}

// List returns work item selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormWorkItemRepository) List(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression, parentExists *bool, start *int, limit *int) ([]WorkItem, int, error) {
	result, count, err := r.listItemsFromDB(ctx, spaceID, criteria, parentExists, start, limit)
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}
	res := make([]WorkItem, len(result))
	for index, value := range result {
		wiType, err := r.witr.LoadTypeFromDB(ctx, value.Type)
		if err != nil {
			return nil, 0, errors.NewInternalError(err.Error())
		}
		modelWI, err := ConvertWorkItemStorageToModel(wiType, &value)
		if err != nil {
			return nil, 0, errors.NewInternalError(err.Error())
		}
		res[index] = *modelWI
	}
	return res, count, nil
}

// Counts returns the amount of work item that satisfy the given criteria.Expression
func (r *GormWorkItemRepository) Count(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression) (int, error) {
	where, parameters, compileError := Compile(criteria)
	if compileError != nil {
		return 0, errors.NewBadParameterError("expression", criteria)
	}
	where = where + " AND space_id = ?"
	parameters = append(parameters, spaceID)

	var count int
	r.db.Model(&WorkItemStorage{}).Where(where, parameters...).Count(&count)
	return count, nil
}

// Fetch fetches the (first) work item matching by the given criteria.Expression.
func (r *GormWorkItemRepository) Fetch(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression) (*WorkItem, error) {
	limit := 1
	results, count, err := r.List(ctx, spaceID, criteria, nil, nil, &limit)
	if err != nil {
		return nil, err
	}
	// if no result
	if count == 0 {
		return nil, nil
	}
	// one result
	result := results[0]
	return &result, nil
}

// GetCountsPerIteration fetches WI count from DB and returns a map of iterationID->WICountsPerIteration
// This function executes following query to fetch 'closed' and 'total' counts of the WI for each iteration in given spaceID
// 	SELECT iterations.id as IterationId, count(*) as Total,
// 		count( case fields->>'system.state' when 'closed' then '1' else null end ) as Closed
// 		FROM "work_items" left join iterations
// 		on fields@> concat('{"system.iteration": "', iterations.id, '"}')::jsonb
// 		WHERE (iterations.space_id = '33406de1-25f1-4969-bcec-88f29d0a7de3'
// 		and work_items.deleted_at IS NULL) GROUP BY IterationId
func (r *GormWorkItemRepository) GetCountsPerIteration(ctx context.Context, spaceID uuid.UUID) (map[string]WICountsPerIteration, error) {
	var res []WICountsPerIteration
	db := r.db.Table("work_items").Select(`iterations.id as IterationId, count(*) as Total,
				count( case fields->>'system.state' when 'closed' then '1' else null end ) as Closed`).Joins(`left join iterations
				on fields@> concat('{"system.iteration": "', iterations.id, '"}')::jsonb`).Where(`iterations.space_id = ?
				and work_items.deleted_at IS NULL`, spaceID).Group(`IterationId`).Scan(&res)
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	countsMap := map[string]WICountsPerIteration{}
	for _, iterationWithCount := range res {
		countsMap[iterationWithCount.IterationID] = iterationWithCount
	}
	return countsMap, nil
}

// GetCountsForIteration returns Closed and Total counts of WI for given iteration
// It executes
// SELECT count(*) as Total, count( case fields->>'system.state' when 'closed' then '1' else null end ) as Closed FROM "work_items" where fields@> concat('{"system.iteration": "%s"}')::jsonb and work_items.deleted_at is null
func (r *GormWorkItemRepository) GetCountsForIteration(ctx context.Context, iterationID uuid.UUID) (map[string]WICountsPerIteration, error) {
	var res WICountsPerIteration
	query := fmt.Sprintf(`SELECT count(*) as Total,
						count( case fields->>'system.state' when 'closed' then '1' else null end ) as Closed
						FROM "work_items"
						where fields@> concat('{"system.iteration": "%s"}')::jsonb
						and work_items.deleted_at is null`, iterationID)
	db := r.db.Raw(query)
	db.Scan(&res)
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	countsMap := map[string]WICountsPerIteration{}
	countsMap[iterationID.String()] = WICountsPerIteration{
		IterationID: iterationID.String(),
		Closed:      res.Closed,
		Total:       res.Total,
	}
	return countsMap, nil
}
