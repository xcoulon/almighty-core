package model

import (
	"reflect"

	"github.com/fabric8-services/fabric8-wit/log"

	"github.com/google/jsonapi"
	uuid "github.com/satori/go.uuid"
)

// RegisterUUIDType register the `uuid.UUID` type in the JSON-API to allow for
// marshalling/unmarshalling `uuid.UUID` in the requests and responses without
// having to deal with manual conversion in our codebase.
func RegisterUUIDType() {
	uuidType := reflect.TypeOf(uuid.UUID{})
	log.Info(nil, map[string]interface{}{
		"type_name":     uuidType.Name(),
		"type_pkg_path": uuidType.PkgPath(),
		"type":          uuidType},
		"registering the `uuid.UUID` type for the JSON-API requests and responses")
	jsonapi.RegisterType(uuidType,
		func(value interface{}) (string, error) {
			return value.(uuid.UUID).String(), nil
		},
		func(value string) (interface{}, error) {
			return uuid.FromString(value)
		})
	uuidPtrType := reflect.TypeOf(&uuid.UUID{})
	log.Info(nil, map[string]interface{}{
		"type_name":     uuidType.Name(),
		"type_pkg_path": uuidType.PkgPath(),
		"type":          uuidType},
		"registering the `&uuid.UUID` type for the JSON-API requests and responses")
	jsonapi.RegisterType(uuidPtrType,
		func(value interface{}) (string, error) {
			if value.(*uuid.UUID) != nil {
				return value.(*uuid.UUID).String(), nil
			}
			return "", nil
		},
		func(value string) (interface{}, error) {
			result, err := uuid.FromString(value)
			if err != nil {
				return nil, err
			}
			return &result, nil
		})
}
