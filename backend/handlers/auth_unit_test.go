package handlers

import "testing"

func TestAuthConfigFromEnvParsesOIDCSuperuserEmails(t *testing.T) {
	t.Setenv("OIDC_SUPERUSER_EMAILS", "alice@example.com, bob@example.com\nCAROL@example.com")

	cfg := AuthConfigFromEnv([]byte("secret"))

	wantEmails := []string{"alice@example.com", "bob@example.com", "CAROL@example.com"}
	if len(cfg.OIDCSuperuserEmails) != len(wantEmails) {
		t.Fatalf("expected %d emails, got %d", len(wantEmails), len(cfg.OIDCSuperuserEmails))
	}
	for i := range wantEmails {
		if cfg.OIDCSuperuserEmails[i] != wantEmails[i] {
			t.Fatalf("email %d: expected %q, got %q", i, wantEmails[i], cfg.OIDCSuperuserEmails[i])
		}
	}
}

func TestIsOIDCSuperuserRequiresVerifiedMatchingEmail(t *testing.T) {
	handler := AuthHandler{Config: AuthConfig{
		OIDCSuperuserEmails: []string{"Admin@Example.com"},
	}}

	if !handler.isOIDCSuperuser(oidcUserClaims{Email: "admin@example.com", EmailVerified: true}) {
		t.Fatal("expected verified matching email to be superuser")
	}

	if handler.isOIDCSuperuser(oidcUserClaims{Email: "admin@example.com", EmailVerified: false}) {
		t.Fatal("expected unverified matching email to be rejected")
	}

	if handler.isOIDCSuperuser(oidcUserClaims{Email: "other@example.com", EmailVerified: true}) {
		t.Fatal("expected non-matching verified email to be rejected")
	}
}
