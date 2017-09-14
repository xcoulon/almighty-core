package token_test

import (
	"context"
	"crypto/rsa"
	"testing"

	"golang.org/x/oauth2"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/resource"
	testtoken "github.com/fabric8-services/fabric8-wit/test/token"
	"github.com/fabric8-services/fabric8-wit/token"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	privateKey   *rsa.PrivateKey
	tokenManager token.Manager
)

func init() {
	privateKey = testtoken.PrivateKey()
	tokenManager = testtoken.NewManager()
}

func TestValidOAuthAccessToken(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	identity := account.Identity{
		ID:       uuid.NewV4(),
		Username: "testuser",
	}
	generatedToken, err := testtoken.GenerateToken(identity.ID.String(), identity.Username, privateKey)
	assert.Nil(t, err)
	accessToken := &oauth2.Token{
		AccessToken: generatedToken,
		TokenType:   "Bearer",
	}

	claims, err := tokenManager.ParseToken(context.Background(), accessToken.AccessToken)
	assert.Nil(t, err)
	assert.Equal(t, identity.ID.String(), claims.Subject)
	assert.Equal(t, identity.Username, claims.Username)
}

func TestInvalidOAuthAccessToken(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	invalidAccessToken := "7423742yuuiy-INVALID-73842342389h"

	accessToken := &oauth2.Token{
		AccessToken: invalidAccessToken,
		TokenType:   "Bearer",
	}

	_, err := tokenManager.ParseToken(context.Background(), accessToken.AccessToken)
	assert.NotNil(t, err)
}

func TestCheckClaimsOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	claims := &token.TokenClaims{
		Email:    "somemail@domain.com",
		Username: "testuser",
	}
	claims.Subject = uuid.NewV4().String()

	assert.Nil(t, token.CheckClaims(claims))
}

func TestCheckClaimsFails(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	claimsNoEmail := &token.TokenClaims{
		Username: "testuser",
	}
	claimsNoEmail.Subject = uuid.NewV4().String()
	assert.NotNil(t, token.CheckClaims(claimsNoEmail))

	claimsNoUsername := &token.TokenClaims{
		Email: "somemail@domain.com",
	}
	claimsNoUsername.Subject = uuid.NewV4().String()
	assert.NotNil(t, token.CheckClaims(claimsNoUsername))

	claimsNoSubject := &token.TokenClaims{
		Email:    "somemail@domain.com",
		Username: "testuser",
	}
	assert.NotNil(t, token.CheckClaims(claimsNoSubject))
}

func TestLocateTokenInContex(t *testing.T) {
	id := uuid.NewV4()

	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["sub"] = id.String()
	ctx := goajwt.WithJWT(context.Background(), tk)

	foundId, err := tokenManager.Locate(ctx)
	require.Nil(t, err)
	assert.Equal(t, id, foundId, "ID in created context not equal")
}

func TestIsServiceAccountTokenInContextClaimPresent(t *testing.T) {
	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["service_accountname"] = "auth"
	ctx := goajwt.WithJWT(context.Background(), tk)

	isServiceAccount := tokenManager.IsServiceAccount(ctx)
	require.True(t, isServiceAccount)
}

func TestIsServiceAccountTokenInContextClaimAbsent(t *testing.T) {
	tk := jwt.New(jwt.SigningMethodRS256)
	ctx := goajwt.WithJWT(context.Background(), tk)

	isServiceAccount := tokenManager.IsServiceAccount(ctx)
	require.False(t, isServiceAccount)
}

func TestIsServiceAccountTokenInContextClaimIncorrect(t *testing.T) {
	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["service_accountname"] = "Not-auth"
	ctx := goajwt.WithJWT(context.Background(), tk)

	isServiceAccount := tokenManager.IsServiceAccount(ctx)
	require.False(t, isServiceAccount)
}

func TestIsServiceAccountTokenInContextClaimIncorrectType(t *testing.T) {
	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["service_accountname"] = 1
	ctx := goajwt.WithJWT(context.Background(), tk)

	isServiceAccount := tokenManager.IsServiceAccount(ctx)
	require.False(t, isServiceAccount)
}

func TestLocateMissingTokenInContext(t *testing.T) {
	ctx := context.Background()

	_, err := tokenManager.Locate(ctx)
	if err == nil {
		t.Error("Should have returned error on missing token in contex", err)
	}
}

func TestLocateMissingUUIDInTokenInContext(t *testing.T) {
	tk := jwt.New(jwt.SigningMethodRS256)
	ctx := goajwt.WithJWT(context.Background(), tk)

	_, err := tokenManager.Locate(ctx)
	require.NotNil(t, err)
}

func TestLocateInvalidUUIDInTokenInContext(t *testing.T) {
	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["sub"] = "131"
	ctx := goajwt.WithJWT(context.Background(), tk)

	_, err := tokenManager.Locate(ctx)
	require.NotNil(t, err)
}

var privateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAnwrjH5iTSErw9xUptp6QSFoUfpHUXZ+PaslYSUrpLjw1q27O
DSFwmhV4+dAaTMO5chFv/kM36H3ZOyA146nwxBobS723okFaIkshRrf6qgtD6coT
HlVUSBTAcwKEjNn4C9jtEpyOl+eSgxhMzRH3bwTIFlLlVMiZf7XVE7P3yuOCpqkk
2rdYVSpQWQWKU+ZRywJkYcLwjEYjc70AoNpjO5QnY+Exx98E30iEdPHZpsfNhsjh
9Z7IX5TrMYgz7zBTw8+niO/uq3RBaHyIhDbvenbR9Q59d88lbnEeHKgSMe2RQpFR
3rxFRkc/64Rn/bMuL/ptNowPqh1P+9GjYzWmPwIDAQABAoIBAQCBCl5ZpnvprhRx
BVTA/Upnyd7TCxNZmzrME+10Gjmz79pD7DV25ejsu/taBYUxP6TZbliF3pggJOv6
UxomTB4znlMDUz0JgyjUpkyril7xVQ6XRAPbGrS1f1Def+54MepWAn3oGeqASb3Q
bAj0Yl12UFTf+AZmkhQpUKk/wUeN718EIY4GRHHQ6ykMSqCKvdnVbMyb9sIzbSTl
v+l1nQFnB/neyJq6P0Q7cxlhVj03IhYj/AxveNlKqZd2Ih3m/CJo0Abtwhx+qHZp
cCBrYj7VelEaGARTmfoIVoGxFGKZNCcNzn7R2ic7safxXqeEnxugsAYX/UmMoq1b
vMYLcaLRAoGBAMqMbbgejbD8Cy6wa5yg7XquqOP5gPdIYYS88TkQTp+razDqKPIU
hPKetnTDJ7PZleOLE6eJ+dQJ8gl6D/dtOsl4lVRy/BU74dk0fYMiEfiJMYEYuAU0
MCramo3HAeySTP8pxSLFYqJVhcTpL9+NQgbpJBUlx5bLDlJPl7auY077AoGBAMkD
UpJRIv/0gYSz5btVheEyDzcqzOMZUVsngabH7aoQ49VjKrfLzJ9WznzJS5gZF58P
vB7RLuIA8m8Y4FUwxOr4w9WOevzlFh0gyzgNY4gCwrzEryOZqYYqCN+8QLWfq/hL
+gYFYpEW5pJ/lAy2i8kPanC3DyoqiZCsUmlg6JKNAoGBAIdCkf6zgKGhHwKV07cs
DIqx2p0rQEFid6UB3ADkb+zWt2VZ6fAHXeT7shJ1RK0o75ydgomObWR5I8XKWqE7
s1dZjDdx9f9kFuVK1Upd1SxoycNRM4peGJB1nWJydEl8RajcRwZ6U+zeOc+OfWbH
WUFuLadlrEx5212CQ2k+OZlDAoGAdsH2w6kZ83xCFOOv41ioqx5HLQGlYLpxfVg+
2gkeWa523HglIcdPEghYIBNRDQAuG3RRYSeW+kEy+f4Jc2tHu8bS9FWkRcsWoIji
ZzBJ0G5JHPtaub6sEC6/ZWe0F1nJYP2KLop57FxKRt0G2+fxeA0ahpMwa2oMMiQM
4GM3pHUCgYEAj2ZjjsF2MXYA6kuPUG1vyY9pvj1n4fyEEoV/zxY1k56UKboVOtYr
BA/cKaLPqUF+08Tz/9MPBw51UH4GYfppA/x0ktc8998984FeIpfIFX6I2U9yUnoQ
OCCAgsB8g8yTB4qntAYyfofEoDiseKrngQT5DSdxd51A/jw7B8WyBK8=
-----END RSA PRIVATE KEY-----`

// RSAPrivateKey returns the key used to sign JWT Tokens
// ssh-keygen -f wit_rsa
func RSAPrivateKey() (*rsa.PrivateKey, error) {
	return parsePrivateKey([]byte(privateKey))
}

// ParsePrivateKey parses a []byte representation of a private key into a rsa.PrivateKey instance
func parsePrivateKey(key []byte) (*rsa.PrivateKey, error) {
	return jwt.ParseRSAPrivateKeyFromPEM(key)
}

var publicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAiRd6pdNjiwQFH2xmNugn
TkVhkF+TdJw19Kpj3nRtsoUe4/6gIureVi7FWqcb+2t/E0dv8rAAs6vl+d7roz3R
SkAzBjPxVW5+hi5AJjUbAxtFX/aYJpZePVhK0Dv8StCPSv9GC3T6bUSF3q3E9R9n
G1SZFkN9m2DhL+45us4THzX2eau6s0bISjAUqEGNifPyYYUzKVmXmHS9fiZJR61h
6TulPwxv68DUSk+7iIJvJfQ3lH/XNWlxWNMMehetcmdy8EDR2IkJCCAbjx9yxgKV
JXdQ7zylRlpaLopock0FGiZrJhEaAh6BGuaoUWLiMEvqrLuyZnJYEg9f/vyxUJSD
JwIDAQAB
-----END PUBLIC KEY-----`

// RSAPublicKey returns the key used to verify JWT Tokens
// openssl rsa -in wit_rsa -pubout -out wit_rsa.pub
func RSAPublicKey() (*rsa.PublicKey, error) {
	return parsePublicKey([]byte(publicKey))
}

// ParsePublicKey parses a []byte representation of a public key into a rsa.PublicKey instance
func parsePublicKey(key []byte) (*rsa.PublicKey, error) {
	return jwt.ParseRSAPublicKeyFromPEM(key)
}
