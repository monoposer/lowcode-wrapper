package auth_test

import (
	"net/http/httptest"
	"testing"

	"github.com/monoposer/dataspan/internal/auth"
)

func TestAdminValidateOpenWhenDisabled(t *testing.T) {
	a := &auth.Admin{}
	if err := a.Validate("anything"); err != auth.ErrAdminAuthDisabled {
		t.Fatalf("err=%v", err)
	}
}

func TestAdminValidateRequiresKey(t *testing.T) {
	a := &auth.Admin{Key: "secret", Enabled: true}
	if err := a.Validate(""); err != auth.ErrAdminMissingKey {
		t.Fatalf("err=%v", err)
	}
}

func TestAdminValidateRejectsWrongKey(t *testing.T) {
	a := &auth.Admin{Key: "secret", Enabled: true}
	if err := a.Validate("wrong"); err != auth.ErrAdminInvalidKey {
		t.Fatalf("err=%v", err)
	}
}

func TestAdminValidateAcceptsKey(t *testing.T) {
	a := &auth.Admin{Key: "secret", Enabled: true}
	if err := a.Validate("secret"); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestAdminAPIKeyFromRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/api/servers", nil)
	req.Header.Set("X-Api-Key", "header-key")
	if got := auth.AdminAPIKeyFromRequest(req); got != "header-key" {
		t.Fatalf("got=%q", got)
	}

	req = httptest.NewRequest("GET", "/admin/api/servers", nil)
	req.Header.Set("Authorization", "Bearer bearer-key")
	if got := auth.AdminAPIKeyFromRequest(req); got != "bearer-key" {
		t.Fatalf("got=%q", got)
	}
}
