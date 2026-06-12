package auth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/monoposer/dataspan/internal/auth"
)

func TestGatewayValidateAnonKey(t *testing.T) {
	g := &auth.Gateway{AnonKey: "anon-secret", Enabled: true}
	_, err := g.Validate("anon-secret", "anon-secret")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGatewayValidateServiceKey(t *testing.T) {
	g := &auth.Gateway{
		AnonKey:    "anon",
		ServiceKey: "service",
		Enabled:    true,
	}
	_, err := g.Validate("service", "service")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGatewayRejectInvalidAPIKey(t *testing.T) {
	g := &auth.Gateway{AnonKey: "anon", Enabled: true}
	_, err := g.Validate("wrong", "anon")
	if err == nil || !errors.Is(err, auth.ErrInvalidAPIKey) {
		t.Fatalf("err=%v", err)
	}
}

func TestGatewayValidateJWT(t *testing.T) {
	secret := []byte("jwt-secret")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "user-1",
		"role": "authenticated",
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}

	g := &auth.Gateway{
		AnonKey:   "anon",
		JWTSecret: secret,
		Enabled:   true,
	}
	claims, err := g.Validate("anon", signed)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-1" || claims.Role != "authenticated" {
		t.Fatalf("claims=%+v", claims)
	}
}

func TestGatewayDisabled(t *testing.T) {
	g := &auth.Gateway{Enabled: false}
	_, err := g.Validate("", "")
	if err != auth.ErrAuthDisabled {
		t.Fatalf("err=%v", err)
	}
}
