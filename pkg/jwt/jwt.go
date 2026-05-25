// Package jwt — signed token issuance & verification.
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type Issuer struct {
	Secret         []byte
	RefreshSecret  []byte
	AccessTTLHrs   int
	RefreshTTLDays int
}

func NewIssuer(secret, refresh string, accessHrs, refreshDays int) *Issuer {
	return &Issuer{
		Secret:         []byte(secret),
		RefreshSecret:  []byte(refresh),
		AccessTTLHrs:   accessHrs,
		RefreshTTLDays: refreshDays,
	}
}

func (i *Issuer) Issue(userID, role string) (access, refresh string, err error) {
	now := time.Now()
	access, err = i.signed(i.Secret, &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(i.AccessTTLHrs) * time.Hour)),
			Subject:   userID,
		},
	})
	if err != nil {
		return
	}
	refresh, err = i.signed(i.RefreshSecret, &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(i.RefreshTTLDays) * 24 * time.Hour)),
			Subject:   userID,
		},
	})
	return
}

func (i *Issuer) Verify(tok string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tok, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return i.Secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (i *Issuer) signed(key []byte, claims *Claims) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(key)
}
