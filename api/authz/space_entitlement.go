package authz

import (
	"context"

	jwt "github.com/dgrijalva/jwt-go"
	contextutils "github.com/fabric8-services/fabric8-wit/api/context_utils"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
)

//NewWorkItemEditorAuthorizator returns a new handler that checks if the current user is allowed to edit the work item
func NewWorkItemEditorAuthorizator(appDB *gormapplication.GormDB, entitlementEndpoint string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		workitemID, err := uuid.FromString(ctx.Param("workitemID")) // the workitem ID param
		if err != nil {
			contextutils.AbortWithError(ctx, err)
			return
		}
		wi, err := appDB.WorkItems().LoadByID(ctx, workitemID)
		if err != nil {
			contextutils.AbortWithError(ctx, err)
			return
		}
		creator := wi.Fields[workitem.SystemCreator]
		if creator == nil {
			contextutils.AbortWithError(ctx, errors.NewBadParameterError("workitem.SystemCreator", nil))
		}
		currentUserID, _ := GetUserID(ctx)
		if currentUserID == nil {
			contextutils.AbortWithError(ctx, errors.NewBadParameterError("workitem.SystemCreator", nil))
		}
		log.Debug(ctx, map[string]interface{}{"wi": wi, "creator": creator, "current_user": currentUserID}, "Authorizing work item update...")
		authorized, err := authorizeWorkitemEditor(ctx, appDB, entitlementEndpoint, wi.SpaceID, creator.(string), currentUserID.String())
		if err != nil {
			contextutils.AbortWithError(ctx, err)
			return
		}
		if !authorized {
			contextutils.AbortWithError(ctx, errors.NewForbiddenError("user is not allowed to update this work item"))
		}
	}
}

// Returns true if the user is the work item creator or a space collaborator
func authorizeWorkitemEditor(ctx *gin.Context, appDB *gormapplication.GormDB, entitlementEndpoint string, spaceID uuid.UUID, creatorID string, editorID string) (bool, error) {
	if editorID == creatorID {
		return true, nil
	}
	tokenPayload, ok := ctx.Get("JWT_PAYLOAD")
	if !ok {
		return false, errors.NewUnauthorizedError("missing token")
	}
	claims, ok := tokenPayload.(jwt.MapClaims)
	if !ok {
		return false, errors.NewUnauthorizedError("invalid token claims type")
	}

	// jwttoken := goajwt.ContextJWT(ctx)
	// tm := tokencontext.ReadTokenManagerFromContext(ctx)
	// if tm == nil {
	// 	log.Error(ctx, map[string]interface{}{
	// 		"token": tm,
	// 	}, "missing token manager")
	// 	return false, errors.NewInternalError(ctx, errs.New("missing token manager"))
	// }
	// tokenWithClaims, err := jwt.ParseWithClaims(jwttoken.Raw, &auth.TokenPayload{}, func(t *jwt.Token) (interface{}, error) {
	// 	return tm.(token.Manager).PublicKey(), nil
	// })
	// if err != nil {
	// 	log.Error(ctx, map[string]interface{}{
	// 		"space_id": spaceID,
	// 		"err":      err,
	// 	}, "unable to parse the rpt token")
	// 	return false, errors.NewInternalError(ctx, errs.Wrap(err, "unable to parse the rpt token"))
	// }
	// claims := tokenWithClaims.Claims.(*auth.TokenPayload)

	// Check if the token was issued before the space resouces changed the last time.
	// If so, we need to re-fetch the rpt token for that space/resource and check permissions.
	outdated, err := isTokenOutdated(ctx, appDB, &claims, entitlementEndpoint, spaceID)
	if err != nil {
		return false, err
	}
	if outdated {
		return checkEntitlementForSpace(ctx, entitlementEndpoint, spaceID)
	}
	authorizationEntry, ok := claims["authorization"]
	if !ok {
		// No authorization in the token. This is not a RPT token. This is an access token.
		// We need to obtain an PRT token.
		log.Warn(ctx, map[string]interface{}{
			"space_id": spaceID,
		}, "no authorization found in the token; this is an access token (not a RPT token)")
		return checkEntitlementForSpace(ctx, entitlementEndpoint, spaceID)
	}
	authorizationPayload, ok := authorizationEntry.(auth.AuthorizationPayload)
	if !ok {
		// No authorization in the token. This is not a RPT token. This is an access token.
		// We need to obtain an PRT token.
		log.Warn(ctx, map[string]interface{}{
			"space_id": spaceID,
		}, "authorization found in the token is not in the expected type")
		return checkEntitlementForSpace(ctx, entitlementEndpoint, spaceID)
	}

	permissions := authorizationPayload.Permissions
	if permissions == nil {
		// if the RPT doesn't contain the resource info, it could be probably
		// because the entitlement was never fetched in the first place. Hence we consider
		// the token to be 'outdated' and hence re-fetch the entitlements from keycloak.
		return checkEntitlementForSpace(ctx, entitlementEndpoint, spaceID)
	}
	for _, permission := range permissions {
		name := permission.ResourceSetName
		if name != nil && spaceID.String() == *name {
			return true, nil
		}
	}
	// if the RPT doesn't contain the resource info, it could be probably
	// because the entitlement was never fetched in the first place. Hence we consider
	// the token to be 'outdated' and hence re-fetch the entitlements from keycloak.
	return checkEntitlementForSpace(ctx, entitlementEndpoint, spaceID)
}

func checkEntitlementForSpace(ctx *gin.Context, entitlementEndpoint string, spaceID uuid.UUID) (bool, error) {
	resource := auth.EntitlementResource{
		Permissions:     []auth.ResourceSet{{Name: spaceID.String()}},
		MetaInformation: auth.EntitlementMeta{Limit: auth.EntitlementLimit},
	}
	userAccessToken := ctx.GetHeader("authorization") // here, the header value will contain the `Bearer ` prefix, but it's ok.
	ent, err := auth.GetEntitlement(ctx, entitlementEndpoint, &resource, userAccessToken)
	if err != nil {
		return false, err
	}
	return ent != nil, nil
}

func isTokenOutdated(ctx context.Context, appDB *gormapplication.GormDB, claims *jwt.MapClaims, entitlementEndpoint string, spaceID uuid.UUID) (bool, error) {
	spaceResource, err := appDB.SpaceResources().LoadBySpace(ctx, &spaceID)
	if err != nil {
		return false, err
	}
	return claims.VerifyIssuedAt(spaceResource.UpdatedAt.Unix(), true), nil
}
