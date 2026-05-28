package jwt

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	m, err := NewManager(Config{})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return m
}

func TestSignAndParseRoundTrip(t *testing.T) {
	m := newTestManager(t)

	token, err := m.Sign(Claims{
		Roles:            "user",
		Platform:         "steam",
		Scope:            []string{"ioi:games"},
		RegisteredClaims: RegisteredClaims{Subject: "profile-123", Issuer: "https://example.test"},
	})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	claims, err := m.Parse(token)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if claims.Subject != "profile-123" {
		t.Errorf("subject = %q, want profile-123", claims.Subject)
	}
	if claims.Platform != "steam" {
		t.Errorf("platform = %q, want steam", claims.Platform)
	}
	if claims.Issuer != "https://example.test" {
		t.Errorf("issuer = %q, want https://example.test", claims.Issuer)
	}
	if claims.ID == "" {
		t.Error("expected a generated jti")
	}
}

func TestSignSetsAtJwtHeader(t *testing.T) {
	m := newTestManager(t)
	token, err := m.Sign(Claims{RegisteredClaims: RegisteredClaims{Subject: "x"}})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	header := strings.SplitN(token, ".", 2)[0]
	raw, err := base64.RawURLEncoding.DecodeString(header)
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal header: %v", err)
	}
	if got["typ"] != "at+jwt" {
		t.Errorf("typ = %v, want at+jwt", got["typ"])
	}
	if got["alg"] != "RS256" {
		t.Errorf("alg = %v, want RS256", got["alg"])
	}
	if got["kid"] != m.Kid() {
		t.Errorf("kid = %v, want %v", got["kid"], m.Kid())
	}
	if got["x5t"] == nil || got["x5t"] == "" {
		t.Error("expected x5t thumbprint in header")
	}
}

func TestParseRejectsWrongKey(t *testing.T) {
	signer := newTestManager(t)
	token, err := signer.Sign(Claims{RegisteredClaims: RegisteredClaims{Subject: "x"}})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	other := newTestManager(t) // different ephemeral key
	if _, err := other.Parse(token); err == nil {
		t.Fatal("expected parse to fail with a different signing key")
	}
}

func TestMiddlewareRejectsMissingToken(t *testing.T) {
	m := newTestManager(t)
	called := false
	h := m.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/profile", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if called {
		t.Error("next handler should not run without a token")
	}
}

func TestMiddlewarePassesValidToken(t *testing.T) {
	m := newTestManager(t)
	token, err := m.Sign(Claims{RegisteredClaims: RegisteredClaims{Subject: "profile-abc"}})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	var gotSubject string
	h := m.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if claims, ok := ClaimsFrom(r.Context()); ok {
			gotSubject = claims.Subject
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if gotSubject != "profile-abc" {
		t.Errorf("subject in context = %q, want profile-abc", gotSubject)
	}
}
