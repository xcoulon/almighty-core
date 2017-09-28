package model

import (
	"github.com/fabric8-services/fabric8-wit/account"
	uuid "github.com/satori/go.uuid"
)

// ConvertUserIdentityToModel converts a business User+Identity into a model User
func ConvertUserIdentityToModel(u account.User, i account.Identity) *User {
	return &User{
		ID:         &i.ID,
		IdentityID: &i.ID,
		UserID:     &u.ID,
		CreatedAt:  formatRFC3339(i.CreatedAt),
		UpdatedAt:  formatRFC3339(i.UpdatedAt),
		Username:   &i.Username,
		Fullname:   &u.FullName,
		ImageURL:   &u.ImageURL,
		Bio:        &u.Bio,
		Company:    &u.Company,
		Email:      &u.Email,
		RegistrationCompleted: &i.RegistrationCompleted,
		URL: &u.URL,
	}
}

type RelatedUser struct {
	ID uuid.UUID `jsonapi:"primary,identities"`
}

type User struct {
	ID                    *uuid.UUID `jsonapi:"primary,identities"`
	IdentityID            *uuid.UUID `jsonapi:"attr,identityID"`
	UserID                *uuid.UUID `jsonapi:"attr,userID"`
	CreatedAt             *string    `jsonapi:"attr,created-at"`
	UpdatedAt             *string    `jsonapi:"attr,updated-at"`
	Username              *string    `jsonapi:"attr,username"`
	Fullname              *string    `jsonapi:"attr,fullname"`
	ImageURL              *string    `jsonapi:"attr,imageURL"`
	Bio                   *string    `jsonapi:"attr,bio"`
	Company               *string    `jsonapi:"attr,company"`
	Email                 *string    `jsonapi:"attr,email"`
	RegistrationCompleted *bool      `jsonapi:"attr,registrationCompleted"`
	URL                   *string    `jsonapi:"attr,url"`
}
