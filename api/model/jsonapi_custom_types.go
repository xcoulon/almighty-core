package model

import (
	"encoding/json"

	"github.com/google/jsonapi"
	uuid "github.com/satori/go.uuid"
)

// UUID the custom UUID type for the JSON-API, which allows us to define a field with the UUID type
// and benefit from implicit conversion to/from string during marshalling/unmarshalling
type UUID struct {
	uuid.UUID
}

// NewUUID returns a new custom-type UUID
func NewUUID(u uuid.UUID) UUID {
	return UUID{u}
}

// MarshalJSON implements the json.Marshaler interface.
func (u UUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (u *UUID) UnmarshalJSON(data []byte) error {
	v, err := uuid.FromBytes(data)
	if err != nil {
		return jsonapi.ErrInvalidType
	}
	u.UUID = v
	return nil
}
