// Package clients · Google + Apple ID-token verification.
//
// Both providers issue signed JWTs from their public JWKs. We verify the
// signature, check audience matches our app, and extract email + sub.
package clients

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type OAuthClaims struct {
	Sub      string
	Email    string
	Name     string
	Picture  string
	Provider string
}

/* ─── Google ─────────────────────────────────────────────── */

type GoogleVerifier struct {
	ClientID string
	jwks     *jwksCache
}

func NewGoogleVerifier(clientID string) *GoogleVerifier {
	return &GoogleVerifier{
		ClientID: clientID,
		jwks:     newJwksCache("https://www.googleapis.com/oauth2/v3/certs"),
	}
}

func (g *GoogleVerifier) Verify(ctx context.Context, idToken string) (*OAuthClaims, error) {
	if g.ClientID == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID not configured")
	}

	tok, err := jwt.Parse(idToken, g.jwks.keyFunc(ctx), jwt.WithIssuedAt(), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("malformed claims")
	}
	iss, _ := claims["iss"].(string)
	if iss != "accounts.google.com" && iss != "https://accounts.google.com" {
		return nil, errors.New("invalid issuer")
	}
	aud, _ := claims["aud"].(string)
	if aud != g.ClientID {
		return nil, errors.New("audience mismatch")
	}
	return &OAuthClaims{
		Sub:      asString(claims["sub"]),
		Email:    asString(claims["email"]),
		Name:     asString(claims["name"]),
		Picture:  asString(claims["picture"]),
		Provider: "google",
	}, nil
}

/* ─── Apple ──────────────────────────────────────────────── */

type AppleVerifier struct {
	BundleID string
	jwks     *jwksCache
}

func NewAppleVerifier(bundleID string) *AppleVerifier {
	return &AppleVerifier{
		BundleID: bundleID,
		jwks:     newJwksCache("https://appleid.apple.com/auth/keys"),
	}
}

func (a *AppleVerifier) Verify(ctx context.Context, idToken string) (*OAuthClaims, error) {
	if a.BundleID == "" {
		return nil, errors.New("APPLE_CLIENT_ID not configured")
	}
	tok, err := jwt.Parse(idToken, a.jwks.keyFunc(ctx), jwt.WithIssuedAt(), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	claims, _ := tok.Claims.(jwt.MapClaims)
	if claims["iss"] != "https://appleid.apple.com" {
		return nil, errors.New("invalid issuer")
	}
	if claims["aud"] != a.BundleID {
		return nil, errors.New("audience mismatch")
	}
	return &OAuthClaims{
		Sub:      asString(claims["sub"]),
		Email:    asString(claims["email"]),
		Provider: "apple",
	}, nil
}

/* ─── JWKS cache ─────────────────────────────────────────── */

type jwksCache struct {
	URL    string
	mu     sync.Mutex
	keys   map[string]*rsa.PublicKey
	fetch  time.Time
	client *http.Client
}

func newJwksCache(url string) *jwksCache {
	return &jwksCache{URL: url, client: &http.Client{Timeout: 5 * time.Second}}
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (c *jwksCache) refresh(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.fetch) < time.Hour && len(c.keys) > 0 {
		return nil
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", c.URL, nil)
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var body struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range body.Keys {
		if k.Kty != "RSA" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		n := new(big.Int).SetBytes(nBytes)
		e := 0
		for _, b := range eBytes {
			e = e<<8 + int(b)
		}
		keys[k.Kid] = &rsa.PublicKey{N: n, E: e}
	}
	c.keys = keys
	c.fetch = time.Now()
	return nil
}

func (c *jwksCache) keyFunc(ctx context.Context) jwt.Keyfunc {
	return func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("token missing kid")
		}
		if err := c.refresh(ctx); err != nil {
			return nil, err
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		key, ok := c.keys[kid]
		if !ok {
			// Force refresh and retry once.
			c.fetch = time.Time{}
			c.mu.Unlock()
			if err := c.refresh(ctx); err != nil {
				c.mu.Lock()
				return nil, err
			}
			c.mu.Lock()
			key, ok = c.keys[kid]
			if !ok {
				return nil, errors.New("unknown kid")
			}
		}
		return key, nil
	}
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

// SplitBearer is a tiny helper so handlers can do `auth := SplitBearer(r.Header)`.
func SplitBearer(h http.Header) string {
	a := h.Get("Authorization")
	if strings.HasPrefix(a, "Bearer ") {
		return strings.TrimPrefix(a, "Bearer ")
	}
	return ""
}
