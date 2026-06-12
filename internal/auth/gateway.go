package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrAuthDisabled     = errors.New("gateway auth disabled")
	ErrMissingAPIKey    = errors.New("missing apikey header")
	ErrInvalidAPIKey    = errors.New("invalid apikey")
	ErrMissingBearer    = errors.New("missing authorization bearer token")
	ErrInvalidJWT       = errors.New("invalid jwt")
)

// Gateway validates inbound apikey + Bearer tokens (Supabase/PostgREST style).
// No RLS: a valid key/JWT grants access to all registered tables.
type Gateway struct {
	AnonKey    string
	ServiceKey string
	JWTSecret  []byte
	Enabled    bool
}

type ctxKey struct{}

// GatewayClaims holds validated JWT claims attached to the request context.
type GatewayClaims struct {
	Subject string
	Role    string
	Raw     jwt.MapClaims
}

func NewGatewayFromEnv() *Gateway {
	anon := strings.TrimSpace(os.Getenv("DATASPAN_ANON_KEY"))
	service := strings.TrimSpace(os.Getenv("DATASPAN_SERVICE_KEY"))
	secret := strings.TrimSpace(os.Getenv("DATASPAN_JWT_SECRET"))

	g := &Gateway{
		AnonKey:    anon,
		ServiceKey: service,
		Enabled:    anon != "",
	}
	if secret != "" {
		g.JWTSecret = []byte(secret)
	}
	return g
}

func (g *Gateway) isAPIKey(key string) bool {
	if key == "" {
		return false
	}
	if g.AnonKey != "" && key == g.AnonKey {
		return true
	}
	if g.ServiceKey != "" && key == g.ServiceKey {
		return true
	}
	return false
}

// Validate checks apikey and Authorization headers.
func (g *Gateway) Validate(apikey, bearer string) (*GatewayClaims, error) {
	if !g.Enabled {
		return nil, ErrAuthDisabled
	}
	apikey = strings.TrimSpace(apikey)
	bearer = strings.TrimSpace(bearer)

	if apikey == "" {
		return nil, ErrMissingAPIKey
	}
	if !g.isAPIKey(apikey) {
		return nil, ErrInvalidAPIKey
	}
	if bearer == "" {
		return nil, ErrMissingBearer
	}
	if g.isAPIKey(bearer) {
		return nil, nil
	}
	if len(g.JWTSecret) == 0 {
		return nil, fmt.Errorf("%w: configure DATASPAN_JWT_SECRET for user tokens", ErrInvalidJWT)
	}
	claims, err := g.parseJWT(bearer)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (g *Gateway) parseJWT(tokenString string) (*GatewayClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("%w: unsupported signing method", ErrInvalidJWT)
		}
		return g.JWTSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJWT, err)
	}
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidJWT
	}
	if exp, err := mapClaims.GetExpirationTime(); err == nil && exp != nil && exp.Before(time.Now()) {
		return nil, fmt.Errorf("%w: token is expired", ErrInvalidJWT)
	}
	out := &GatewayClaims{Raw: mapClaims}
	if sub, err := mapClaims.GetSubject(); err == nil {
		out.Subject = sub
	}
	if role, _ := mapClaims["role"].(string); role != "" {
		out.Role = role
	}
	return out, nil
}

func BearerToken(authorization string) string {
	authorization = strings.TrimSpace(authorization)
	const prefix = "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
}

func WithGatewayClaims(ctx context.Context, claims *GatewayClaims) context.Context {
	if claims == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, claims)
}

func GatewayClaimsFrom(ctx context.Context) *GatewayClaims {
	c, _ := ctx.Value(ctxKey{}).(*GatewayClaims)
	return c
}
