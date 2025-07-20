package authentication

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaim struct {
	Value map[string]string `json:"value,omitempty"`
	jwt.RegisteredClaims
}

type JWT struct {
	secret []byte
}

func NewJWT(key string) (*JWT, error) {
	return &JWT{
		secret: []byte(key),
	}, nil
}
func GenerateKey(length int) (string, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return "", errors.Join(errors.New("err when generating secret key"), err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

func (j *JWT) NewToken(value map[string]string, d time.Duration) (string, error) {
	claims := CustomClaim{
		Value: value,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(d)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	result, err := token.SignedString(j.secret)
	if err != nil {
		return "", errors.Join(errors.New("err when signing the token"), err)
	}
	return result, nil
}
func (j *JWT) Parse(input string) (map[string]string, error) {
	token, err := jwt.ParseWithClaims(input, &CustomClaim{}, func(t *jwt.Token) (interface{}, error) {
		//check if the token signature is valid
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signature")
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, errors.Join(errors.New("err when parsing token"), err)
	}
	if claims, ok := token.Claims.(*CustomClaim); ok {
		return claims.Value, nil
	} else {
		return nil, errors.New("invalid claim")
	}
}
