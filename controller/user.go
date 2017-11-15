package controller

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

// UserController implements the user resource.
type UserController struct {
	*goa.Controller
	config     UserControllerConfiguration
	InitTenant func(context.Context) error
}

// UserControllerConfiguration the configuration for the UserController
type UserControllerConfiguration interface {
	auth.AuthServiceConfiguration
	GetCacheControlUser() string
}

// NewUserController creates a user controller.
func NewUserController(service *goa.Service, config UserControllerConfiguration) *UserController {
	return &UserController{
		Controller: service.NewController("UserController"),
		config:     config,
	}
}

// Show returns the authorized user based on the provided Token
func (c *UserController) Show(ctx *app.ShowUserContext) error {
	client, err := auth.CreateAuthClient(ctx, c.config)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	client.SetJWTSigner(goasupport.NewForwardSigner(ctx))
	res, err := client.ShowUser(goasupport.ForwardContextRequestID(ctx), authservice.ShowUserPath(), ctx.IfModifiedSince, ctx.IfNoneMatch)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to get user from the auth service")
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "unable to get user from the auth service"))
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case 200:
	// OK
	case 401:
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(rest.ReadBody(res.Body)))
	default:
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Errorf("status: %s, body: %s", res.Status, rest.ReadBody(res.Body))))
	}

	authUser, _ := client.DecodeUser(res)

	if c.InitTenant != nil {
		go func(ctx context.Context) {
			c.InitTenant(ctx)
		}(ctx)
	}
	return ctx.OK(convertToAppUser(ctx.RequestData, authUser))
}

func convertToAppUser(request *goa.RequestData, user *authservice.User) *app.User {
	return &app.User{
		Data: &app.UserData{
			ID:   user.Data.ID,
			Type: user.Data.Type,
			Attributes: &app.UserDataAttributes{
				CreatedAt:             user.Data.Attributes.CreatedAt,
				UpdatedAt:             user.Data.Attributes.UpdatedAt,
				Username:              user.Data.Attributes.Username,
				FullName:              user.Data.Attributes.FullName,
				ImageURL:              user.Data.Attributes.ImageURL,
				Bio:                   user.Data.Attributes.Bio,
				URL:                   user.Data.Attributes.URL,
				UserID:                user.Data.Attributes.UserID,
				IdentityID:            user.Data.Attributes.IdentityID,
				ProviderType:          user.Data.Attributes.ProviderType,
				Email:                 user.Data.Attributes.Email,
				Company:               user.Data.Attributes.Company,
				ContextInformation:    user.Data.Attributes.ContextInformation,
				RegistrationCompleted: user.Data.Attributes.RegistrationCompleted,
			},
			Links: &app.GenericLinks{
				Self:    user.Data.Links.Self,
				Related: user.Data.Links.Related,
				Meta:    user.Data.Links.Meta,
			},
		},
	}
}
