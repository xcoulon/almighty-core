package token

import (
	"crypto/rsa"
	"fmt"

	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/account"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Manager generate and find auth token information
type Manager interface {
	Extract(string) (*account.Identity, error)
	Locate(ctx context.Context) (uuid.UUID, error)
	PublicKey() *rsa.PublicKey
}

type tokenManager struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

// NewManager returns a new token Manager for handling tokens
func NewManager(publicKey *rsa.PublicKey) Manager {
	return &tokenManager{
		publicKey: publicKey,
	}
}

// NewManagerWithPrivateKey returns a new token Manager for handling creation of tokens with both private and pulic keys
func NewManagerWithPrivateKey(privateKey *rsa.PrivateKey) Manager {
	return &tokenManager{
		publicKey:  &privateKey.PublicKey,
		privateKey: privateKey,
	}
}

func (mgm tokenManager) Extract(tokenString string) (*account.Identity, error) {
	fmt.Printf("Extracting from '%s'\n", tokenString)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return mgm.publicKey, nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if !token.Valid {
		return nil, errors.New("Token not valid")
	}

	claimedUUID := token.Claims.(jwt.MapClaims)["sub"]
	if claimedUUID == nil {
		return nil, errors.New("Subject can not be nil")
	}
	// in case of nil UUID, below type casting will fail hence we need above check
	id, err := uuid.FromString(token.Claims.(jwt.MapClaims)["sub"].(string))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ident := account.Identity{
		ID:       id,
		Username: token.Claims.(jwt.MapClaims)["preferred_username"].(string),
	}

	return &ident, nil
}

func (mgm tokenManager) Locate(ctx context.Context) (uuid.UUID, error) {
	token := goajwt.ContextJWT(ctx)
	if token == nil {
		return uuid.UUID{}, errors.New("Missing token") // TODO, make specific tokenErrors
	}
	id := token.Claims.(jwt.MapClaims)["sub"]
	if id == nil {
		return uuid.UUID{}, errors.New("Missing sub")
	}
	idTyped, err := uuid.FromString(id.(string))
	if err != nil {
		return uuid.UUID{}, errors.New("uuid not of type string")
	}
	return idTyped, nil
}

func (mgm tokenManager) PublicKey() *rsa.PublicKey {
	return mgm.publicKey
}
