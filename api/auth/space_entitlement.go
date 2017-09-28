package authz

import (
	"context"
	"fmt"
	"strings"

	contextutils "github.com/fabric8-services/fabric8-wit/api/context_utils"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
	jwt "gopkg.in/dgrijalva/jwt-go.v3"
)

//NewWorkItemEditorAuthorizator returns a new handler that checks if the current user is allowed to edit the work item
func NewWorkItemEditorAuthorizator(appDB *gormapplication.GormDB, config *configuration.ConfigurationData) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		log.Warn(ctx, nil, "authorizing user...")
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
			return
		}
		currentUserID, _ := GetUserID(ctx)
		if currentUserID == nil {
			contextutils.AbortWithError(ctx, errors.NewBadParameterError("workitem.SystemCreator", nil))
			return
		}
		log.Debug(ctx, map[string]interface{}{"wi": wi, "creator": creator, "current_user": currentUserID}, "Authorizing work item update...")
		entitlementEndpoint, err := config.GetKeycloakEndpointEntitlement(ctx.Request)
		if err != nil {
			contextutils.AbortWithError(ctx, errors.NewInternalError(ctx, err))
			return
		}

		authorized, err := authorizeWorkitemEditor(ctx, appDB, entitlementEndpoint, wi.SpaceID, creator.(string), currentUserID.String())
		if err != nil {
			contextutils.AbortWithError(ctx, err)
			return
		}
		if !authorized {
			contextutils.AbortWithError(ctx, errors.NewForbiddenError("user is not allowed to update this work item"))
			return
		}
	}
}

// Returns true if the user is the work item creator or a space collaborator
func authorizeWorkitemEditor(ctx *gin.Context, appDB *gormapplication.GormDB, entitlementEndpoint string, spaceID uuid.UUID, creatorID string, currentUserID string) (bool, error) {
	log.Warn(ctx, nil, "checking workitem editor...")
	if currentUserID == creatorID {
		return true, nil
	}
	tokenPayload, found := ctx.Get("JWT_PAYLOAD")
	if !found {
		log.Error(ctx,
			map[string]interface{}{"user_id": currentUserID},
			"missing token")
		return false, errors.NewUnauthorizedError("missing token")
	}
	claims, isMapClaims := tokenPayload.(jwt.MapClaims)
	if !isMapClaims {
		log.Error(ctx,
			map[string]interface{}{"user_id": currentUserID},
			"invalid token claims type: %T", tokenPayload)
		return false, errors.NewUnauthorizedError(fmt.Sprintf("invalid token claims type: %T", tokenPayload))
	}

	// Check if the token was issued before the space resouces changed the last time.
	// If so, we need to re-fetch the rpt token for that space/resource and check permissions.
	outdated, err := isTokenOutdated(ctx, appDB, &claims, entitlementEndpoint, spaceID)
	if err != nil {
		log.Error(ctx,
			map[string]interface{}{
				"user_id": currentUserID,
				"error":   err.Error()},
			"failed to check if token is outdated")
		return false, err
	}
	if outdated {
		return checkEntitlementForSpace(ctx, entitlementEndpoint, spaceID)
	}
	authorizationEntry, ok := claims["authorization"]
	if !ok {
		// No authorization in the token. This is not a RPT token. This is an access token.
		// We need to obtain an PRT token.
		log.Debug(ctx, map[string]interface{}{
			"space_id": spaceID,
		}, "no authorization found in the token; this is an access token (not a RPT token)")
		return checkEntitlementForSpace(ctx, entitlementEndpoint, spaceID)
	}
	authorizationPayload, ok := authorizationEntry.(token.AuthorizationPayload)
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
		log.Debug(ctx, map[string]interface{}{}, "checking entitlement as there's no permission in the token's `authorization` claim...")
		return checkEntitlementForSpace(ctx, entitlementEndpoint, spaceID)
	}

	log.Warn(ctx, map[string]interface{}{
		"space_id":    spaceID,
		"permissions": permissions,
	}, "checking token's `authorization` claim for permissions on the work item's space")
	for _, permission := range permissions {
		name := permission.ResourceSetName
		if name != nil && spaceID.String() == *name {
			return true, nil
		}
	}
	log.Warn(ctx, map[string]interface{}{
		"space_id":    spaceID,
		"permissions": permissions,
	}, "no permission found in the token's `authorization` claim for permissions")

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
	authorizationHeader := ctx.GetHeader("Authorization") // here, the header value will contain the `Bearer ` prefix, but it's ok.
	userAccessToken := strings.Replace(authorizationHeader, "Bearer ", "", 1)
	ent, err := auth.GetEntitlement(ctx, entitlementEndpoint, &resource, userAccessToken)
	if err != nil {
		log.Warn(ctx, map[string]interface{}{"space_id": spaceID, "error": err.Error()}, "user has no entitlement for this space")
		return false, err
	}
	log.Warn(ctx, map[string]interface{}{"space_id": spaceID, "entitlement": ent}, "user entitlement for this space ?")
	return ent != nil, nil
}

func isTokenOutdated(ctx context.Context, appDB *gormapplication.GormDB, claims *jwt.MapClaims, entitlementEndpoint string, spaceID uuid.UUID) (bool, error) {
	spaceResource, err := appDB.SpaceResources().LoadBySpace(ctx, &spaceID)
	if err != nil {
		return false, err
	}
	return claims.VerifyIssuedAt(spaceResource.UpdatedAt.Unix(), true), nil
}
