package area_test

import (
	"strconv"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/path"
	"github.com/pkg/errors"

	localerror "github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestAreaRepository struct {
	gormtestsupport.DBTestSuite

	clean func()
}

func TestRunAreaRepository(t *testing.T) {
	suite.Run(t, &TestAreaRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (test *TestAreaRepository) SetupTest() {
	test.clean = cleaner.DeleteCreatedEntities(test.DB)
}

func (test *TestAreaRepository) TearDownTest() {
	test.clean()
}

func (test *TestAreaRepository) TestCreateAreaWithSameNameFail() {
	t := test.T()

	resource.Require(t, resource.Database)

	repo := area.NewAreaRepository(test.DB)

	name := "Area 21"
	newSpace := space.Space{
		Name: "Space 1 " + uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	assert.Nil(t, err)

	i := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}

	repo.Create(context.Background(), &i)
	if i.ID == uuid.Nil {
		t.Errorf("Area was not created, ID nil")
	}

	require.False(t, i.CreatedAt.After(time.Now()), "Area was not created, CreatedAt after Now()")

	assert.Equal(t, name, i.Name)

	anotherAreaWithSameName := area.Area{
		Name:    i.Name,
		SpaceID: space.ID,
	}
	err = repo.Create(context.Background(), &anotherAreaWithSameName)
	assert.NotNil(t, err)

	// In case of unique constrain error, a BadParameterError is returned.
	_, ok := errors.Cause(err).(localerror.BadParameterError)
	assert.True(t, ok)
}

func (test *TestAreaRepository) TestCreateArea() {
	t := test.T()

	resource.Require(t, resource.Database)

	repo := area.NewAreaRepository(test.DB)

	name := "Area 21"
	newSpace := space.Space{
		Name: uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(t, err)

	i := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}

	repo.Create(context.Background(), &i)
	if i.ID == uuid.Nil {
		t.Errorf("Area was not created, ID nil")
	}

	if i.CreatedAt.After(time.Now()) {
		t.Errorf("Area was not created, CreatedAt after Now()?")
	}

	assert.Equal(t, name, i.Name)
}

func (test *TestAreaRepository) TestCreateChildArea() {
	t := test.T()

	resource.Require(t, resource.Database)

	repo := area.NewAreaRepository(test.DB)

	newSpace := space.Space{
		Name: uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(t, err)

	name := "Area #24"
	name2 := "Area #24.1"

	i := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	err = repo.Create(context.Background(), &i)
	assert.Nil(t, err)

	// ltree field doesnt accept "-" , so we will save them as "_"
	expectedPath := path.Path{i.ID}
	area2 := area.Area{
		Name:    name2,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	assert.Nil(t, err)

	actualArea, err := repo.Load(context.Background(), area2.ID)
	actualPath := actualArea.Path
	require.Nil(t, err)
	require.NotNil(t, actualArea)
	assert.Equal(t, expectedPath, actualPath)

}

func (test *TestAreaRepository) TestListAreaBySpace() {
	t := test.T()

	resource.Require(t, resource.Database)

	repo := area.NewAreaRepository(test.DB)

	newSpace := space.Space{
		Name: uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space1, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(t, err)

	var createdAreaIds []uuid.UUID
	for i := 0; i < 3; i++ {
		name := "Test Area #20" + strconv.Itoa(i)

		a := area.Area{
			Name:    name,
			SpaceID: space1.ID,
		}
		err := repo.Create(context.Background(), &a)
		assert.Equal(t, nil, err)
		createdAreaIds = append(createdAreaIds, a.ID)
		t.Log(a.ID)
	}

	newSpace2 := space.Space{
		Name: uuid.NewV4().String(),
	}
	space2, err := repoSpace.Create(context.Background(), &newSpace2)
	require.Nil(t, err)

	err = repo.Create(context.Background(), &area.Area{
		Name:    "Other Test area #20",
		SpaceID: space2.ID,
	})
	assert.Equal(t, nil, err)

	its, err := repo.List(context.Background(), space1.ID)
	assert.Nil(t, err)
	assert.Len(t, its, 3)

	for i := 0; i < 3; i++ {
		assert.NotNil(t, searchInAreaSlice(createdAreaIds[i], its))
	}
}

func searchInAreaSlice(searchKey uuid.UUID, areaList []*area.Area) *area.Area {
	for i := 0; i < len(areaList); i++ {
		if searchKey == areaList[i].ID {
			return areaList[i]
		}
	}
	return nil
}

func (test *TestAreaRepository) TestListChildrenOfParents() {
	t := test.T()
	resource.Require(t, resource.Database)
	repo := area.NewAreaRepository(test.DB)

	name := "Area #240"
	name2 := "Area #240.1"
	name3 := "Area #240.2"
	var createdAreaIDs []uuid.UUID

	newSpace := space.Space{
		Name: uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(t, err)

	// *** Create Parent Area ***
	i := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	err = repo.Create(context.Background(), &i)
	require.Nil(t, err)

	// *** Create 1st child area ***

	// ltree field doesnt accept "-" , so we will save them as "_"
	expectedPath := path.Path{i.ID}
	area2 := area.Area{
		Name:    name2,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	require.Nil(t, err)
	createdAreaIDs = append(createdAreaIDs, area2.ID)

	actualArea, err := repo.Load(context.Background(), area2.ID)
	actualPath := actualArea.Path
	require.Nil(t, err)
	assert.NotEqual(t, uuid.Nil, area2.Path)
	assert.Equal(t, expectedPath, actualPath) // check that path ( an ltree field ) was populated.

	// *** Create 2nd child area ***

	expectedPath = path.Path{i.ID}
	area3 := area.Area{
		Name:    name3,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area3)
	require.Nil(t, err)
	createdAreaIDs = append(createdAreaIDs, area3.ID)

	actualArea, err = repo.Load(context.Background(), area3.ID)
	require.Nil(t, err)

	actualPath = actualArea.Path
	assert.Equal(t, expectedPath, actualPath)

	// *** Validate that there are 2 children
	childAreaList, err := repo.ListChildren(context.Background(), &i)
	require.Nil(t, err)

	assert.Equal(t, 2, len(childAreaList))

	for i := 0; i < len(createdAreaIDs); i++ {
		assert.NotNil(t, createdAreaIDs[i], childAreaList[i].ID)
	}
}

func (test *TestAreaRepository) TestListImmediateChildrenOfGrandParents() {
	t := test.T()

	resource.Require(t, resource.Database)
	repo := area.NewAreaRepository(test.DB)

	name := "Area #240"
	name2 := "Area #240.1"
	name3 := "Area #240.1.3"

	newSpace := space.Space{
		Name: uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(t, err)

	// *** Create Parent Area ***

	i := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	err = repo.Create(context.Background(), &i)
	assert.Nil(t, err)

	// *** Create 'son' area ***

	expectedPath := path.Path{i.ID}
	area2 := area.Area{
		Name:    name2,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	require.Nil(t, err)

	childAreaList, err := repo.ListChildren(context.Background(), &i)
	assert.Equal(t, 1, len(childAreaList))
	require.Nil(t, err)

	// *** Create 'grandson' area ***

	expectedPath = path.Path{i.ID, area2.ID}
	area4 := area.Area{
		Name:    name3,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area4)
	require.Nil(t, err)

	childAreaList, err = repo.ListChildren(context.Background(), &i)

	// But , There is only 1 'son' .

	require.Nil(t, err)
	assert.Equal(t, 1, len(childAreaList))
	assert.Equal(t, area2.ID, childAreaList[0].ID)

	// *** Confirm the grandson has no son

	childAreaList, err = repo.ListChildren(context.Background(), &area4)
	assert.Equal(t, 0, len(childAreaList))
}

func (test *TestAreaRepository) TestListParentTree() {
	t := test.T()

	resource.Require(t, resource.Database)
	repo := area.NewAreaRepository(test.DB)

	name := "Area #240"
	name2 := "Area #240.1"

	newSpace := space.Space{
		Name: uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(t, err)
	// *** Create Parent Area ***
	i := area.Area{
		Name:    name,
		SpaceID: newSpace.ID,
	}
	err = repo.Create(context.Background(), &i)
	assert.Nil(t, err)

	// *** Create 'son' area ***
	expectedPath := path.Path{i.ID}
	area2 := area.Area{
		Name:    name2,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	require.Nil(t, err)

	listOfCreatedID := []uuid.UUID{i.ID, area2.ID}
	listOfCreatedAreas, err := repo.LoadMultiple(context.Background(), listOfCreatedID)

	require.Nil(t, err)
	assert.Equal(t, 2, len(listOfCreatedAreas))

	for i := 0; i < 2; i++ {
		assert.NotNil(t, searchInAreaSlice(listOfCreatedID[i], listOfCreatedAreas))
	}

}
