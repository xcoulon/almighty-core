package handler_test

import (
	"context"
	"fmt"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

func createOneRandomUserIdentity(ctx context.Context, db *gorm.DB, username string) *account.Identity {
	newUserUUID := uuid.NewV4()
	identityRepo := account.NewIdentityRepository(db)
	identity := account.Identity{
		Username: username,
		ID:       newUserUUID,
	}
	err := identityRepo.Create(ctx, &identity)
	if err != nil {
		fmt.Println("should not happen off.")
		return nil
	}
	return &identity
}
