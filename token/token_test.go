package token_test

import (
	"crypto/rsa"
	"testing"
	"time"

	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/resource"
	testtoken "github.com/fabric8-services/fabric8-wit/test/token"
	"github.com/fabric8-services/fabric8-wit/token"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestExtractToken(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	manager := createManager(t)

	identity := account.Identity{
		ID:       uuid.NewV4(),
		Username: "testuser",
	}
	privateKey, err := RSAPrivateKey()
	if err != nil {
		t.Fatal("Could not parse private key", err)
	}

	token, err := testtoken.GenerateToken(identity.ID.String(), identity.Username, privateKey)
	if err != nil {
		t.Fatal("Could not generate test token", err)
	}

	ident, err := manager.Extract(token)
	if err != nil {
		t.Fatal("Could not extract Identity from generated token", err)
	}
	assert.Equal(t, identity.Username, ident.Username)
}

func TestExtractWithInvalidToken(t *testing.T) {
	// This tests generates invalid Token
	// by setting expired date, empty UUID, not setting UUID
	// all above cases are invalid
	// hence manager.Extract should fail in all above cases
	manager := createManager(t)
	privateKey, err := RSAPrivateKey()

	tok := jwt.New(jwt.SigningMethodRS256)
	// add already expired time to "exp" claim"
	claims := jwt.MapClaims{"sub": "some_uuid", "exp": float64(time.Now().Unix() - 100)}
	tok.Claims = claims
	tokenStr, err := tok.SignedString(privateKey)
	if err != nil {
		panic(err)
	}
	idn, err := manager.Extract(tokenStr)
	if err == nil {
		t.Error("Expired token should not be parsed. Error must not be nil", idn, err)
	}

	// now set correct EXP but do not set uuid
	claims = jwt.MapClaims{"exp": float64(time.Now().AddDate(0, 0, 1).Unix())}
	tok.Claims = claims
	tokenStr, err = tok.SignedString(privateKey)
	if err != nil {
		panic(err)
	}
	idn, err = manager.Extract(tokenStr)
	if err == nil {
		t.Error("Invalid token should not be parsed. Error must not be nil", idn, err)
	}

	// now set UUID to empty String
	claims = jwt.MapClaims{"sub": ""}
	tok.Claims = claims
	tokenStr, err = tok.SignedString(privateKey)
	if err != nil {
		panic(err)
	}
	idn, err = manager.Extract(tokenStr)
	if err == nil {
		t.Error("Invalid token should not be parsed. Error must not be nil", idn, err)
	}
}

func TestLocateTokenInContex(t *testing.T) {
	id := uuid.NewV4()

	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["sub"] = id.String()
	ctx := goajwt.WithJWT(context.Background(), tk)

	manager := createManager(t)

	foundId, err := manager.Locate(ctx)
	if err != nil {
		t.Error("Failed not locate token in given context", err)
	}
	assert.Equal(t, id, foundId, "ID in created context not equal")
}

func TestLocateMissingTokenInContext(t *testing.T) {
	ctx := context.Background()

	manager := createManager(t)

	_, err := manager.Locate(ctx)
	if err == nil {
		t.Error("Should have returned error on missing token in contex", err)
	}
}

func TestLocateMissingUUIDInTokenInContext(t *testing.T) {
	tk := jwt.New(jwt.SigningMethodRS256)
	ctx := goajwt.WithJWT(context.Background(), tk)

	manager := createManager(t)

	_, err := manager.Locate(ctx)
	if err == nil {
		t.Error("Should have returned error on missing token in contex", err)
	}
}

func TestLocateInvalidUUIDInTokenInContext(t *testing.T) {
	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["sub"] = "131"
	ctx := goajwt.WithJWT(context.Background(), tk)

	manager := createManager(t)

	_, err := manager.Locate(ctx)
	if err == nil {
		t.Error("Should have returned error on missing token in contex", err)
	}
}

func createManager(t *testing.T) token.Manager {
	privateKey, err := RSAPrivateKey()
	if err != nil {
		t.Fatal("Could not parse private key")
	}

	return token.NewManagerWithPrivateKey(privateKey)
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
