//go:build unit || !integration

package objectstore_test

import (
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type ObjectStoreSuite struct {
	suite.Suite
	db *objectstore.DB
}

func TestObjectStoreSuite(t *testing.T) {
	suite.Run(t, new(ObjectStoreSuite))
}

func (s *ObjectStoreSuite) SetupTest() {
	s.db = objectstore.NewObjectStore(
		objectstore.WithLocation(":memory:"),
	)
}

func (s *ObjectStoreSuite) TeardownTest() {
	s.db.Close()
}

type Payload struct {
	ID   string
	Name string
	Age  int
}

type DBPayload struct {
	Payload
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (DBPayload) TableName() string {
	return "payloads"
}

func (s *ObjectStoreSuite) TestSimple() {
	err := s.db.Connect(&DBPayload{})
	require.NoError(s.T(), err)

	p := DBPayload{
		Payload: Payload{
			ID:   "1",
			Name: "Adam",
			Age:  1,
		},
	}

	err = s.db.Create(&p)
	require.NoError(s.T(), err)

	retrievedP := DBPayload{}
	found := s.db.Get(&retrievedP, "1")
	require.True(s.T(), found)

	retrievedP.Age = retrievedP.Age + 1
	err = s.db.Save(&retrievedP)
	require.NoError(s.T(), err)

	newRetrievedP := DBPayload{}
	found = s.db.Get(&newRetrievedP, "1")
	require.True(s.T(), found)
	require.Equal(s.T(), 2, newRetrievedP.Age)

	newRetrievedP.ID = "2"
	err = s.db.Save(&newRetrievedP)
	require.NoError(s.T(), err)

	var payloads []Payload
	err = s.db.GetBy(&payloads, "Age > ?", 0)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, len(payloads))
}
